# Git Hooks for nself

This directory contains example git hooks to enhance security and prevent accidental commits of sensitive data.

## Available Hooks

### pre-commit.example

Prevents committing sensitive files to git, including:
- `.env.secrets` - Production secrets
- `.env.local` - Local environment overrides
- SSL private keys (`*.key`, `*.pem`)
- SSH private keys (`*_rsa`, `*id_rsa*`)
- Certificate files (`*.p12`, `*.pfx`)

The hook also checks:
- `.env.secrets` file permissions (should be 600)
- `.env.secrets` is in .gitignore

## Installation

### Manual Installation

1. Copy the example hook to your project's `.git/hooks/` directory:
   ```bash
   cp .github/hooks/pre-commit.example .git/hooks/pre-commit
   chmod +x .git/hooks/pre-commit
   ```

2. Test the hook:
   ```bash
   # Try to commit .env.secrets (should fail)
   git add .env.secrets
   git commit -m "test"
   ```

### Automated Installation (Coming Soon)

```bash
nself config hooks install
```

## Bypassing the Hook (NOT RECOMMENDED)

If you absolutely need to bypass the hook (e.g., for testing), use:

```bash
git commit --no-verify
```

**WARNING:** Only use `--no-verify` if you fully understand the security implications!

## Customization

You can customize the hook by editing `.git/hooks/pre-commit`:

1. Add additional file patterns to block
2. Add custom security checks
3. Modify warning messages

## Security Best Practices

1. **Never commit secrets to git** - Use `.env.secrets` and keep it gitignored
2. **Set secure permissions** - `chmod 600 .env.secrets`
3. **Use the hook** - Install the pre-commit hook on all development machines
4. **Review commits** - Always review staged changes before committing
5. **Rotate secrets** - If secrets are accidentally committed, rotate them immediately

## Troubleshooting

### Hook not running

- Ensure the hook is executable: `chmod +x .git/hooks/pre-commit`
- Check that the hook file is named exactly `pre-commit` (no extension)

### Hook blocking legitimate commits

- Review the `SENSITIVE_FILES` array in the hook
- Remove patterns that don't apply to your project
- Remember: the hook is there to protect you!

### False positives

If the hook incorrectly blocks a file:
1. Review why it's being blocked
2. Ensure the file doesn't contain actual secrets
3. If safe, adjust the pattern matching in the hook

## Additional Resources

- [Git Hooks Documentation](https://git-scm.com/docs/githooks)
- [nself Security Guide](../../docs/guides/SECURITY.md)
- [Environment Variables Guide](../../docs/configuration/ENVIRONMENT-VARIABLES.md)
