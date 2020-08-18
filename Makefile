# Makefile for the bundlechanges library.

PROJECT := github.com/juju/bundlechanges

default: check

check:
	go test $(PROJECT)/...

clean:
	go clean $(PROJECT)/...

format:
	gofmt -w -l .

help:
	@echo -e 'Juju Bundle Changes - list of make targets:\n'
	@echo 'make check - Run tests.'
	@echo 'make clean - Remove object files from package source directories.'
	@echo 'make format - Format the source files.'

.PHONY: check clean format help
