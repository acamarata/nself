# Governance — nSelf

## Decision Model

nSelf uses a **BDFL** (Benevolent Dictator For Life) model. The project founder, [Aric Camarata](mailto:aric.camarata@gmail.com), holds final decision authority over the roadmap, architecture, and community standards.

Day-to-day code review authority is **delegated** to the CODEOWNERS listed in `.github/CODEOWNERS`. CODEOWNERS have merge rights on their assigned areas and are expected to exercise judgment within the project's established patterns.

## Delegation

- CODEOWNERS review and merge pull requests in their designated areas.
- The founder reviews and merges changes to governance, security, licensing, pricing, and major architectural decisions.
- Any contributor can open a pull request. Merge requires approval from the relevant CODEOWNER.

## Evolution Path

The BDFL model is appropriate for the current project size. When the project reaches **~50 active contributors** or has **2-3 named external committers** with a consistent track record of quality contributions, governance will evolve to a **maintainer council** model.

Under the maintainer council model, a small group of trusted maintainers (3-7 people) makes decisions by consensus, with the founder serving as a tiebreaker. The threshold for council formation is documented here, not left vague, so contributors can track progress.

## RFC Process

A lightweight RFC (Request for Comments) process for significant changes is planned for a future release. Until then, major proposals should be opened as GitHub issues with the `proposal` label and discussed before implementation begins. The founder or a CODEOWNER will tag the issue `accepted`, `needs-revision`, or `declined` after discussion.


## DCO Sign-Off

nSelf uses the [Developer Certificate of Origin (DCO)](https://developercertificate.org/) instead of a CLA. All contributors must sign off on their commits:

```bash
git commit -s -m "feat: your change description"
```

The `-s` flag appends `Signed-off-by: Your Name <you@example.com>` to your commit message.

The DCO is lighter-weight than a CLA: you are certifying that you have the right to submit the contribution and that it can be distributed under the project's MIT license. No paperwork, no corporate approval needed.

## Community Standards

- All contributors are expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
- Security issues are handled through the process in [SECURITY.md](SECURITY.md).
- Code of conduct violations are handled through the process in [ENFORCEMENT.md](ENFORCEMENT.md).

## Contact

- General questions: [GitHub Discussions](https://github.com/nself-org/cli/discussions)
- Security reports: see [SECURITY.md](SECURITY.md)
- Code of conduct issues: conduct@nself.org
- Founder: aric.camarata@gmail.com
