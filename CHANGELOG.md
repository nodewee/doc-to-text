# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0]

### Changed
- Changed command-line parameter `--llm_template` to `--llm-template` for consistency with naming conventions

### Improved
- **Major Architecture Refactor**: Complete restructuring of core components for better maintainability
- Enhanced cross-platform compatibility and path handling
- Streamlined configuration management with environment variable overrides
- Improved validation for input/output paths and file operations
- Improved progress tracking with two-tier logging system (critical vs detailed)

## [0.3.0]

### Added
- Core document text extraction functionality
- Support for multiple file formats (PDF, images, e-books, office documents, HTML, text files)
- OCR capabilities with Surya OCR and LLM Caller integration
- Content-type strategy selection (text-first vs image-first processing)
- Interactive tool selection with auto-detection
- Cross-platform compatibility (Windows, macOS, Linux)
- Resume capability for interrupted large document processing
- Fallback extraction chains for robust processing
- Comprehensive error handling and retry mechanisms
- Structured logging with progress indicators
- Build system with version injection and multi-platform binaries 