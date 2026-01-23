import sys
import subprocess
import importlib.util

def install(package):
    subprocess.check_call([sys.executable, "-m", "pip", "install", package])

# Check if PyMuPDF is installed and install it if not
if importlib.util.find_spec('fitz') is None:
    print("The required 'PyMuPDF' library is not installed. Attempting to install...")
    install('pymupdf')

import fitz  # Now this should work as PyMuPDF is installed

def extract_text_from_pdf(pdf_path):
    document = fitz.open(pdf_path)
    text = ""
    for page_num in range(len(document)):
        page = document.load_page(page_num)
        text += page.get_text()
    return text

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python try_extract_pdf_text.py <path_to_pdf>")
        sys.exit(1)

    pdf_path = sys.argv[1]
    extracted_text = extract_text_from_pdf(pdf_path)
    print(extracted_text)
