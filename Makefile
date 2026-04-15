.PHONY: build install uninstall clean

BINARY = macsetgo
INSTALL_PATH = /usr/local/bin/$(BINARY)

build:
	go build -ldflags="-s -w" -o $(BINARY) .

install: build
	sudo cp $(BINARY) $(INSTALL_PATH)
	sudo chmod +x $(INSTALL_PATH)
	@echo "Installed to $(INSTALL_PATH)"

uninstall:
	sudo rm -f $(INSTALL_PATH)
	@echo "Uninstalled $(BINARY)"

clean:
	rm -f $(BINARY)
