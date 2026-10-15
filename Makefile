# The showcase is built outside the repo, so tmux and a Ghostty window
# outside tmux run the same binary (docs/walkthrough.md).
SHOWCASE ?= /tmp/showcase

.PHONY: showcase

# Build the showcase from the working tree and open it.
showcase:
	go build -o $(SHOWCASE) ./examples/showcase
	$(SHOWCASE)
