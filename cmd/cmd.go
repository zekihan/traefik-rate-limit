package main

type command string

const (
	// CommandServer is the command to run the server
	CommandServer command = "server"
	// CommandClient is the command to run the client
	CommandClient command = "client"
	// CommandHealthCheck is the command to run the health check
	CommandHealthCheck command = "healthcheck"
)
