## What does this PR do?

<!-- Brief description of the change. Link the issue if there is one: Fixes #123 -->

## Checklist

- [ ] Tests added for new behavior (`bats src/tests/` passes)
- [ ] ShellCheck clean (`shellcheck -S error src/cli/*.sh src/lib/**/*.sh`)
- [ ] No Bash 4+ syntax (verified with `grep -rn 'echo -e\|declare -A\|mapfile\|readarray' src/`)
- [ ] Docs updated (`.wiki/commands/` help text updated, `COMMAND-TREE-V1.md` updated if structure changed)
- [ ] CI passes on Bash 3.2 (macOS default)

## Testing

<!-- How did you test this? Commands run, output observed. -->

```bash
# Commands I ran to verify:

```
