# Documentation Tooling

The OVASABI project uses Yarn for managing JavaScript dependencies related to documentation
formatting and validation.

## Setup

Initial setup is required to use the documentation tools:

```bash
# Option 1: Run the setup script
./setup-yarn.sh

# Option 2: Use the make command
make js-setup
```

This will install Yarn if needed, set up the Yarn environment, and install all necessary
dependencies.

## Using Documentation Tools

The project includes several commands to help maintain documentation quality:

```bash
# Check for formatting issues (without fixing)
make lint

# Fix formatting issues automatically
make lint-fix
```

## Package Management with Yarn

Yarn manages all dependencies in a local `node_modules` directory which is automatically excluded
from git. Benefits of using Yarn include:

1. **Faster Installation**: Parallel downloads and caching
2. **Deterministic Builds**: Precise versioning via yarn.lock
3. **Improved Security**: Checksums validation
4. **Offline Mode**: Installing packages without internet connection

## Tooling Details

The project uses these documentation tools:

- **Prettier**: For consistent formatting of Markdown files
- **markdown-link-check**: For validating links in documentation

## Configuration

- `.prettierrc.json`: Configuration for Prettier
- `.prettierignore`: Files/directories excluded from Prettier formatting
- `.yarnrc.yml`: Yarn configuration
- `package.json`: NPM package configuration

## Adding New Documentation Tools

To add new documentation tooling dependencies:

```bash
# Add a new development dependency
yarn add --dev new-package-name
```

## Troubleshooting

If you encounter issues with documentation tooling:

1. Ensure Yarn is properly installed: `yarn --version`
2. Try reinstalling dependencies: `rm -rf node_modules && yarn install`
3. Check for Yarn errors: `yarn --verbose`
