BIN_NAME := f2fs-extractor
OUT_DIR := out

# Terminal Colors
GREEN := \033[0;32m
CYAN  := \033[0;36m
NC    := \033[0m

.PHONY: all clean linux darwin windows __build-linux __build-darwin __build-windows __verify __final_all

all: clean __build-linux __build-darwin __build-windows __verify __final_all

linux: __build-linux __verify
	@echo -e ""
	@echo -e "$(GREEN) ✓ Linux builds completed successfully!$(NC)"

darwin: __build-darwin __verify
	@echo -e ""
	@echo -e "$(GREEN) ✓ macOS builds completed successfully!$(NC)"

windows: __build-windows __verify
	@echo -e ""
	@echo -e "$(GREEN) ✓ Windows builds completed successfully!$(NC)"

clean:
	@rm -rf $(OUT_DIR)
	@echo -e "$(CYAN)Output directory cleaned!$(NC)"
	@echo -e ""

__build-linux:
	@echo -e "$(CYAN)Building for Linux...$(NC)"
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/linux/x86_64/$(BIN_NAME) .
	GOOS=linux GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/linux/x86/$(BIN_NAME) .
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/linux/armeabi-v7a/$(BIN_NAME) .
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/linux/arm64-v8a/$(BIN_NAME) .
	@echo -e ""

__build-darwin:
	@echo -e "$(CYAN)Building for macOS (Darwin)...$(NC)"
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/darwin/x86_64/$(BIN_NAME) .
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/darwin/arm64-v8a/$(BIN_NAME) .
	@echo -e ""

__build-windows:
	@echo -e "$(CYAN)Building for Windows...$(NC)"
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/windows/x86_64/$(BIN_NAME).exe .
	GOOS=windows GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/windows/x86/$(BIN_NAME).exe .
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/windows/arm64-v8a/$(BIN_NAME).exe .
	@echo -e ""

__verify:
	@echo -e "$(CYAN)Generated Binaries in $(OUT_DIR)/:$(NC)"
	@find $(OUT_DIR) -type f | sort | while read -r file; do \
		size=$$(du -h "$$file" | cut -f1); \
		printf "  %-45s %s\n" "$$file" "$$size"; \
	done

__final_all:
	@echo -e ""
	@echo -e "$(GREEN) ✓ All builds completed successfully for $(BIN_NAME)!$(NC)"
