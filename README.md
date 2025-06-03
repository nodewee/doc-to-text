# Doc Text Extractor

A CLI tool for extracting text from various document formats with configurable OCR capabilities.

## ✨ Features

- **Multi-format Support**: PDF, Word, HTML, E-books, Images, and text files
- **Configurable OCR**: LLM Caller (AI-powered) or Surya OCR (fast & multilingual)
- **Smart Content Strategy**: Choose between text-first or image-first processing for PDFs
- **Interactive Tool Selection**: Auto-detects available tools and prompts for selection
- **Persistent Configuration**: Auto-detection and JSON-based tool path management
- **Cross-platform**: macOS, Linux, Windows with automatic tool detection

## 📚 Documentation

- **🚀 [Quick Start](QUICKSTART.md)** - Installation and basic usage
- **🔧 [Development](DEVELOPMENT.md)** - Architecture and development guide
- **🏷️ [Versioning](VERSIONING.md)** - Version management and build system

## 🛠️ Installation

### Install Dependencies

```bash
# macOS
brew install ghostscript pandoc calibre && pip install surya-ocr

# Ubuntu/Linux  
sudo apt-get install ghostscript pandoc calibre && pip install surya-ocr

# Windows (with Chocolatey)
choco install ghostscript pandoc calibre && pip install surya-ocr
```

Download pre-built binaries from [Releases](../../releases) or build from source with `go build`.

## 🔧 Basic Usage

```bash
# Extract with interactive selection (prompts for OCR tool and content type)
doc-to-text document.pdf

# Use specific OCR tool
doc-to-text document.pdf --ocr surya_ocr
doc-to-text document.pdf --ocr llm-caller --llm_template qwen-vl-ocr

# Specify content processing strategy for PDFs
doc-to-text document.pdf --content-type text    # Try Calibre first, OCR fallback
doc-to-text document.pdf --content-type image   # Direct OCR processing

# Custom output
doc-to-text document.pdf -o output.txt

# Version info
doc-to-text --version    # Quick version
doc-to-text version      # Detailed build info
```

## ⚙️ Configuration

The tool auto-detects available tools on first run and saves paths to `~/.doc-to-text/config.json`.

### Configuration Commands

```bash
# Manage tool paths
doc-to-text config list
doc-to-text config get surya_ocr_path
doc-to-text config set surya_ocr_path /custom/path/surya_ocr
```

### Environment Variable Overrides

```bash
# Temporarily override settings
DOC_TEXT_OCR_STRATEGY=surya_ocr doc-to-text document.pdf
DOC_TEXT_CONTENT_TYPE=text doc-to-text document.pdf
DOC_TEXT_MAX_CONCURRENCY=8 doc-to-text document.pdf
SURYA_OCR_PATH=/custom/path doc-to-text document.pdf
```

### Key Configuration Options

| Setting | Description | Default |
|---------|-------------|---------|
| `ocr_strategy` | OCR tool selection | `interactive` |
| `content_type` | PDF processing strategy | `image` |
| `max_concurrency` | Concurrent processes | `4` |
| `verbose` | Enable progress output | `false` |

## 📁 Supported Formats

| Type | Extensions | Method |
|------|------------|--------|
| **PDFs** | `.pdf` | OCR or Calibre (based on content-type) |
| **Images** | `.jpg`, `.png`, `.gif`, `.bmp`, `.tiff` | OCR |
| **Documents** | `.doc`, `.docx`, `.rtf`, `.odt`, `.ppt`, `.xls` | Pandoc |
| **Web** | `.html`, `.mhtml` | Built-in parser |
| **E-books** | `.epub`, `.mobi` | Calibre |
| **Text** | `.txt`, `.md`, `.json`, `.csv`, `.xml`, `.py`, `.js` | Direct reading |

## 🔧 OCR Engines

### Surya OCR (Recommended)
- **Fast** and **multilingual** (100+ languages)
- **Installation**: `pip install surya-ocr`
- **Best for**: Standard documents, batch processing

### LLM Caller (Configurable AI)
- **AI-powered** with template-based approach
- **Requires**: `--llm_template` parameter
- **Best for**: Complex layouts, handwritten text, specific models

### Interactive Selection
- **Default mode**: Automatically prompts for tool selection
- **Smart detection**: Shows only available engines
- **Auto-selection**: For text content-type, automatically selects best tool without prompts

## 💡 Key Concepts

### Content Type Strategy

The `--content-type` parameter determines PDF processing strategy:

- **`text`**: Tries Calibre first (fast for text-based PDFs), then OCR if failed
- **`image`**: Uses OCR directly (default, best for scanned documents)

### Output Organization

Text is extracted to organized directories:
- Input: `/path/to/document.pdf`  
- Output: `/path/to/{md5_hash}/text.txt`
- Pages: `/path/to/{md5_hash}/pages/` (for PDFs)

### Resume Capability

Large document processing can be interrupted and resumed. The tool automatically:
- Detects completed pages and skips them
- Continues from the last processed page
- Maintains processing state in intermediate directories

## 🚨 Common Issues

**OCR tool not found**: Check tool installation and use `doc-to-text config set <tool>_path /path/to/tool`

**Permission errors**: Ensure tools are executable and paths are accessible

**Poor OCR quality**: Try different OCR engines or ensure good source quality (300 DPI recommended)

## 📄 License

[MIT License](LICENSE)