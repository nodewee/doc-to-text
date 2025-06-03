# Quick Start Guide

Get started with Doc Text Extractor in 3 minutes.

## 🚀 Quick Install

```bash
# macOS
brew install ghostscript pandoc calibre && pip install surya-ocr

# Linux
sudo apt-get install ghostscript pandoc calibre && pip install surya-ocr

# Windows (with Chocolatey)
choco install ghostscript pandoc calibre && pip install surya-ocr
```

Download the binary from [Releases](../../releases) or build with `go build`.

## 📖 Basic Usage

### Simple Extraction

```bash
# Interactive mode (prompts for OCR tool and content strategy)
doc-to-text document.pdf

# Specific OCR tool
doc-to-text document.pdf --ocr surya_ocr
doc-to-text document.pdf --ocr llm-caller --llm_template qwen-vl-ocr

# Content processing strategy
doc-to-text document.pdf --content-type text    # Try Calibre first, OCR fallback
doc-to-text document.pdf --content-type image   # Direct OCR (default)

# Custom output
doc-to-text document.pdf -o extracted_text.txt
```

### File Type Examples

```bash
# PDFs (scanned or text-based)
doc-to-text document.pdf --ocr surya_ocr

# Images
doc-to-text screenshot.png -o text_output.txt

# E-books
doc-to-text novel.epub

# Office documents
doc-to-text report.docx
```

## ⚙️ Configuration

### Auto-Detection

On first run, the tool automatically:
- Detects available OCR engines
- Saves tool paths to `~/.doc-to-text/config.json`
- Creates organized output directories

### Basic Configuration

```bash
# View current settings
doc-to-text config list

# Set tool paths
doc-to-text config set surya_ocr_path /custom/path/surya_ocr

# Environment overrides
DOC_TEXT_OCR_STRATEGY=surya_ocr doc-to-text document.pdf
DOC_TEXT_CONTENT_TYPE=text doc-to-text document.pdf
```

## 🔧 OCR Tools

### Surya OCR (Recommended)
- **Installation**: `pip install surya-ocr`
- **Best for**: Fast processing, multilingual support
- **Usage**: `--ocr surya_ocr`

### LLM Caller (AI-Powered)
- **Installation**: Follow llm-caller setup guide
- **Best for**: Complex layouts, configurable AI models
- **Usage**: `--ocr llm-caller --llm_template qwen-vl-ocr`

### Interactive Mode
- **Default**: Prompts to choose available tools
- **Smart**: Auto-selects for text content type

## 💡 Key Features

### Content Type Strategy

Choose PDF processing approach:
- **`--content-type text`**: Fast Calibre extraction with OCR fallback
- **`--content-type image`**: Direct OCR processing (default)

### Resume Processing

Large PDFs process page-by-page with automatic resume:
```bash
doc-to-text large_document.pdf --ocr surya_ocr
# If interrupted, run again to continue from last page
```

### Output Organization

Files are organized by MD5 hash:
```
input_directory/
├── document.pdf
└── a1b2c3d4.../          # MD5 hash of document.pdf
    ├── text.txt          # Final extracted text
    └── pages/            # Individual page files (PDFs)
```

## 🚨 Troubleshooting

**Tool not found**: Use `doc-to-text config set <tool>_path /path/to/tool`

**Permission errors**: Ensure tools are executable (`chmod +x /path/to/tool`)

**Poor quality**: Try different OCR tools or ensure good source resolution (300 DPI)

**Batch processing**: Use environment variables for consistent settings:
```bash
for file in *.pdf; do
    DOC_TEXT_OCR_STRATEGY=surya_ocr doc-to-text "$file"
done
```

## 📚 Next Steps

- **Architecture**: [DEVELOPMENT.md](DEVELOPMENT.md) for development info
- **Versioning**: [VERSIONING.md](VERSIONING.md) for build and release details
- **Full Documentation**: [README.md](README.md) for complete usage guide 