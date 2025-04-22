package main

type command string

const (
	// CommandServer is the command to run the server
	CommandServer command = "server"
	// CommandClient is the command to run the client
	CommandClient command = "client"
	// CommandHealthCheck is the command to run the health check
	CommandHealthCheck command = "healthcheck"
	// CommandVersion is the command to run the version check
	CommandVersion command = "version"
	// CommandHelp is the command to show help
	CommandHelp command = "help"
)
