# Contributing to SSH-X-Term

Thank you for your interest in contributing!  
SSH-X-Term is an open-source project and welcomes contributions of all kinds.

> By contributing, you agree any code or documentation you submit is your own work and does not create any liability for the maintainers regarding credential or password safety.

## How to Contribute

- **Bug Reports & Feature Requests:**  
  Please use [GitHub Issues](https://github.com/eugeniofciuvasile/ssh-x-term/issues) for bugs, suggestions, or feature requests.

- **Pull Requests:**  
  1. Fork the repo and create your branch from `main`.
  2. Make your changes, following the style of existing code.
  3. Write concise, meaningful commit messages.
  4. Add or update tests as appropriate.
  5. Open a pull request with a clear description of your changes.

- **Coding Guidelines:**  
  - Write concise comments at the top of functions and switch blocks.
  - Use Bubble Tea-compatible log helpers for visible feedback.
  - For security, never log or store plaintext secrets.

- **Development Prerequisites:**  
  - Go 1.24+
  - Bitwarden CLI (`bw`)
  - sshpass (Unix) or plink.exe (Windows)
  - tmux (recommended for best experience)

- **Testing:**  
  Please test your changes on your OS and, if possible, across both Unix and Windows environments.

## Code of Conduct

By participating, you agree to abide by the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/).

---

Thank you for making SSH-X-Term better!
