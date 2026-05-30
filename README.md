# ⚓ gh-complain

Create a GitHub issue from a specific local code snippet directly from your terminal in **one single command**. No browser clicking, no context switching, no manual formatting.

---

## 📦 Installation

This tool runs as a native extension for the official **GitHub CLI (`gh`)**. 

1. Make sure you have the GitHub CLI installed and logged in:
```bash
gh auth login
Install this extension directly from GitHub:
```

```bash
gh extension install srivatsachadaga-collab/gh-complain
```
🚀 Usage
Navigate into any active Git repository on your machine and run:

```bash
gh complain <file_path> <start_line>-<end_line> "<issue_title>"
```

📄 License
This project is licensed under the MIT License - see the LICENSE file for details.

---