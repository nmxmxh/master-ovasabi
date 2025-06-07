# mds_to_pdf.sh Documentation

## Overview

`mds_to_pdf.sh` is a utility script designed to automate the conversion of all Markdown (`.md`)
files in the `mds/` directory into PDF format. The script stores the resulting PDFs in a dedicated
`pdf/` directory and appends a specified date to each PDF filename. For enhanced security and
integrity, it also generates a SHA256 cryptographic checksum for each PDF file.

## Features

- **Batch Conversion:** Converts all `.md` files in `mds/` to PDF using Pandoc.
- **Organized Output:** Stores all PDFs in a `pdf/` directory at the project root.
- **Date-Stamped Filenames:** Appends the date `2025-06-04` to each PDF filename for versioning and
  traceability.
- **Cryptographic Enhancement:** Generates a SHA256 checksum for each PDF, saved as a `.sha256` file
  alongside the PDF for integrity verification.

## Prerequisites

- [Pandoc](https://pandoc.org/) must be installed (`brew install pandoc` on macOS).
- (Optional, recommended) A LaTeX engine such as `xelatex` or `pdflatex` for best PDF output
  (`brew install --cask mactex` on macOS).

## Usage

1. Place the script in your project root.
2. Make it executable:
   ```bash
   chmod +x mds_to_pdf.sh
   ```
3. Run the script:
   ```bash
   ./mds_to_pdf.sh
   ```

## Output

- All PDFs will be saved in the `pdf/` directory, named as `<original>_2025-06-04.pdf`.
- Each PDF will have a corresponding `<original>_2025-06-04.pdf.sha256` file containing its SHA256
  hash.

## Example

If you have `mds/example.md`, after running the script you will get:

- `pdf/example_2025-06-04.pdf`
- `pdf/example_2025-06-04.pdf.sha256`

## Security & Integrity

The SHA256 checksum allows you (or any recipient) to verify the integrity of each PDF. To check a
file:

```bash
shasum -a 256 -c pdf/example_2025-06-04.pdf.sha256
```

## Customization

- To change the date, edit the `DATE` variable in the script.
- To process subdirectories or add more cryptographic features (e.g., digital signatures), modify
  the script as needed.

## License

This script is provided as-is, without warranty. You are free to modify and distribute it within
your project.
