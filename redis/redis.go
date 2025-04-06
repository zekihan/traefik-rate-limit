package redis

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

func NewClient(opts *Options) *Client {
	conn, err := net.Dial("tcp", opts.Addr)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to Redis: %v", err))
	}

	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

type Options struct {
	Addr string
}

func (c *Client) Ping(_ context.Context) *Cmd {
	_, err := fmt.Fprint(c.conn, "PING\r\n")
	if err != nil {
		return &Cmd{err: err}
	}

	response, err := c.reader.ReadString('\n')
	if err != nil {
		return &Cmd{err: err}
	}

	return &Cmd{val: strings.TrimSpace(response)}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Eval implements the EVAL command for Redis Lua scripts
func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd {
	cmdArgs := []string{"EVAL", script, strconv.Itoa(len(keys))}
	cmdArgs = append(cmdArgs, keys...)

	for _, arg := range args {
		cmdArgs = append(cmdArgs, fmt.Sprint(arg))
	}

	return c.executeCommand(ctx, cmdArgs)
}

// EvalSha implements the EVALSHA command for Redis Lua scripts
func (c *Client) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd {
	cmdArgs := []string{"EVALSHA", sha1, strconv.Itoa(len(keys))}
	cmdArgs = append(cmdArgs, keys...)

	for _, arg := range args {
		cmdArgs = append(cmdArgs, fmt.Sprint(arg))
	}

	return c.executeCommand(ctx, cmdArgs)
}

// ScriptExists checks if the script exists in Redis script cache
func (c *Client) ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd {
	cmdArgs := []string{"SCRIPT", "EXISTS"}
	cmdArgs = append(cmdArgs, hashes...)

	cmd := c.executeCommand(ctx, cmdArgs)
	return &BoolSliceCmd{cmd: cmd}
}

// ScriptLoad loads a script into the Redis script cache
func (c *Client) ScriptLoad(ctx context.Context, script string) *StringCmd {
	cmd := c.executeCommand(ctx, []string{"SCRIPT", "LOAD", script})
	return &StringCmd{cmd: cmd}
}

// Del deletes keys from Redis
func (c *Client) Del(ctx context.Context, keys ...string) *IntCmd {
	cmdArgs := []string{"DEL"}
	cmdArgs = append(cmdArgs, keys...)

	cmd := c.executeCommand(ctx, cmdArgs)
	return &IntCmd{cmd: cmd}
}

// FlushDB deletes all keys from the current database
func (c *Client) FlushDB(ctx context.Context) *Cmd {
	return c.executeCommand(ctx, []string{"FLUSHDB"})
}

// EvalRO executes a read-only script
func (c *Client) EvalRO(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd {
	// In Redis 7+, we could use EVAL_RO, but for compatibility we'll just use EVAL
	return c.Eval(ctx, script, keys, args...)
}

// EvalShaRO executes a read-only script by sha
func (c *Client) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd {
	// In Redis 7+, we could use EVALSHA_RO, but for compatibility we'll just use EVALSHA
	return c.EvalSha(ctx, sha1, keys, args...)
}

// executeCommand sends a command to Redis and reads the response
func (c *Client) executeCommand(ctx context.Context, args []string) *Cmd {
	cmd := &Cmd{}

	// Add a context timeout if present
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 {
			err := c.conn.SetDeadline(deadline)
			if err != nil {
				cmd.err = err
				return cmd
			}
		}
	}

	// Build Redis protocol command
	command := buildRedisCommand(args)
	_, err := c.conn.Write([]byte(command))
	if err != nil {
		cmd.err = err
		return cmd
	}

	// Read response
	resp, err := readRedisResponse(c.reader)
	if err != nil {
		cmd.err = err
		return cmd
	}

	cmd.val = resp
	return cmd
}

// buildRedisCommand creates a Redis protocol command from args
func buildRedisCommand(args []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*%d\r\n", len(args)))

	for _, arg := range args {
		b.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}

	return b.String()
}

// readRedisResponse reads a response from Redis
func readRedisResponse(reader *bufio.Reader) (interface{}, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if len(line) < 2 {
		return nil, fmt.Errorf("invalid response line: %q", line)
	}

	switch line[0] {
	case '+': // Simple string
		return strings.TrimSpace(line[1:]), nil
	case '-': // Error
		return nil, fmt.Errorf("redis error: %s", strings.TrimSpace(line[1:]))
	case ':': // Integer
		return strconv.ParseInt(strings.TrimSpace(line[1:]), 10, 64)
	case '$': // Bulk string
		length, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil // Redis nil
		}

		data := make([]byte, length+2) // +2 for CRLF
		_, err = reader.Read(data)
		if err != nil {
			return nil, err
		}
		return string(data[:length]), nil
	case '*': // Array
		count, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return nil, err
		}
		if count == -1 {
			return nil, nil // Redis nil
		}

		array := make([]interface{}, count)
		for i := 0; i < count; i++ {
			item, err := readRedisResponse(reader)
			if err != nil {
				return nil, err
			}
			array[i] = item
		}
		return array, nil
	default:
		return nil, fmt.Errorf("unknown response prefix: %c", line[0])
	}
}
