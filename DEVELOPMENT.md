# Development Guide

Development guide for the Doc Text Extractor project.

## 🏗️ Project Structure

```
doc-to-text/
├── cmd/                    # CLI commands
├── pkg/
│   ├── config/            # Runtime configuration
│   ├── constants/         # Platform constants
│   ├── core/              # Business logic
│   ├── providers/         # Format extractors
│   └── ocr/               # OCR system
├── main.go                # Entry point
└── build.sh               # Build script
```

## 🔧 Core Components

- **Configuration**: Environment-based runtime configuration
- **OCR Processing**: Page-by-page PDF processing with multiple engines
- **Extractor System**: Factory pattern with fallback chains

## 🚀 Development Setup

```bash
git clone <repository-url>
cd doc-to-text
go mod download
./build.sh local
```

## 📋 Code Guidelines

- Write code and comments in English
- Keep functions focused and simple
- Use structured error handling
- Follow Go naming conventions

## 📊 Logging System

### Logging Levels and Usage

The application uses a structured logging system with different levels for various use cases:

#### **ProgressAlways()** - Essential Progress Information
- **Purpose**: Critical progress updates that users should always see
- **When to use**: Major processing milestones, key status changes
- **Visibility**: Always displayed regardless of verbose mode
- **Examples**:
  - OCR engine selection
  - File processing start/completion
  - Text extraction completion with timing
  - Cache loading notifications

#### **Progress()** - Detailed Progress Information  
- **Purpose**: Detailed progress tracking for debugging and monitoring
- **When to use**: Step-by-step processing details, intermediate steps
- **Visibility**: Only displayed in verbose mode (`-v` flag)
- **Examples**:
  - Page-by-page processing details
  - Directory creation notifications
  - Individual file operations
  - Cache hit details

#### **Info()** - System Information
- **Purpose**: System status and configuration details
- **When to use**: Initialization info, configuration details, debug info
- **Visibility**: Only displayed in verbose mode (`-v` flag)
- **Examples**:
  - Processor initialization
  - Configuration settings
  - File analysis details
  - Extractor registration

#### **Debug()** - Development Information
- **Purpose**: Detailed debugging information for development
- **When to use**: Internal operations, detailed error context
- **Visibility**: Only displayed in debug mode
- **Examples**:
  - Command execution details
  - Internal state changes
  - Detailed error contexts

### User Experience Examples

#### Default Mode (no `-v` flag)
```
🔍 Attempting extraction with ocr (attempt 1/1)
🔍 Using OCR engine: surya_ocr
✅ Primary extractor 'ocr' succeeded!
💾 Text saved to: output.txt
✅ Text extraction completed successfully in 1234ms
```

#### Verbose Mode (with `-v` flag)
```
📋 === Starting file processing ===
📂 Input file: document.pdf
📤 Output file: output.txt
[INFO] File processor initialized with configuration:
[INFO] Runtime settings applied from environment and command line
[INFO] File analysis completed:
[INFO]   Extension: pdf
[INFO]   MIME type: application/pdf
🔍 Attempting extraction with ocr (attempt 1/1)
📄 Using page-by-page processing for better control
📂 Created pages directory: /tmp/pages
🔍 Using OCR engine: surya_ocr
✂️ Splitting PDF into individual pages...
✅ Successfully split PDF into 5 pages
📄 Processing page 1/5
📄 Processing page 2/5
...
✅ Primary extractor 'ocr' succeeded!
💾 Text saved to: output.txt
✅ Text extraction completed successfully in 1234ms
```

### Implementation Guidelines

When adding new log messages:

1. **Choose the appropriate level**:
   - Use `ProgressAlways()` for user-facing milestones
   - Use `Progress()` for detailed step tracking
   - Use `Info()` for system information
   - Use `Debug()` for development details

2. **Include emoji icons** for better visual distinction:
   - 🔍 Processing/searching
   - ✅ Success/completion
   - ⚠️ Warnings
   - ❌ Errors
   - 📂 File operations
   - 💾 Save operations
   - 🔄 Progress indicators
   ```

3. **Maintain consistency** across different extractors and processors 