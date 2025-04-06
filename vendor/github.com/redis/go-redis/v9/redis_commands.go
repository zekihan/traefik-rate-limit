package redis

import (
	"context"
	"errors"
)

// Script represents a Redis Lua script
type Script struct {
	src string
}

// NewScript creates a new Redis script
func NewScript(src string) *Script {
	return &Script{src: src}
}

// Run executes the script
func (s *Script) Run(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	return c.Eval(ctx, s.src, keys, args...)
}

// Redis command types

// Cmd represents a Redis command
type Cmd struct {
	val interface{}
	err error
}

// Err returns the error of the Redis command
func (c *Cmd) Err() error {
	return c.err
}

// Result returns the value and error of the Redis command
func (c *Cmd) Result() (interface{}, error) {
	return c.val, c.err
}

// StringCmd represents a Redis command that returns a string
type StringCmd struct {
	cmd *Cmd
}

// Result returns the string value and error of the Redis command
func (c *StringCmd) Result() (string, error) {
	if c.cmd.err != nil {
		return "", c.cmd.err
	}

	s, ok := c.cmd.val.(string)
	if !ok {
		return "", errors.New("redis: expected string result")
	}
	return s, nil
}

// IntCmd represents a Redis command that returns an integer
type IntCmd struct {
	cmd *Cmd
}

// Result returns the integer value and error of the Redis command
func (c *IntCmd) Result() (int64, error) {
	if c.cmd.err != nil {
		return 0, c.cmd.err
	}

	i, ok := c.cmd.val.(int64)
	if !ok {
		return 0, errors.New("redis: expected integer result")
	}
	return i, nil
}

// Err returns the error of the Redis command
func (c *IntCmd) Err() error {
	return c.cmd.err
}

// BoolSliceCmd represents a Redis command that returns a slice of booleans
type BoolSliceCmd struct {
	cmd *Cmd
}

// Result returns the boolean slice and error of the Redis command
func (c *BoolSliceCmd) Result() ([]bool, error) {
	if c.cmd.err != nil {
		return nil, c.cmd.err
	}

	arr, ok := c.cmd.val.([]interface{})
	if !ok {
		return nil, errors.New("redis: expected array result")
	}

	bools := make([]bool, len(arr))
	for i, v := range arr {
		switch val := v.(type) {
		case int64:
			bools[i] = val != 0
		case string:
			bools[i] = val == "1"
		default:
			return nil, errors.New("redis: unexpected result type for SCRIPT EXISTS")
		}
	}
	return bools, nil
}

// SliceCmd represents a Redis command that returns a slice of values
type SliceCmd struct {
	cmd *Cmd
}

// Result returns the slice and error of the Redis command
func (c *SliceCmd) Result() ([]interface{}, error) {
	if c.cmd.err != nil {
		return nil, c.cmd.err
	}

	arr, ok := c.cmd.val.([]interface{})
	if !ok {
		return nil, errors.New("redis: expected array result")
	}
	return arr, nil
}

// Redis is an interface for Redis client
type Redis interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd
	ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *StringCmd
	Del(ctx context.Context, keys ...string) *IntCmd
	FlushDB(ctx context.Context) *Cmd

	EvalRO(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd
	EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd

	Close() error
	Ping() *Cmd
}

type Scripter interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd
	EvalRO(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd
	EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd
	ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *StringCmd
}
