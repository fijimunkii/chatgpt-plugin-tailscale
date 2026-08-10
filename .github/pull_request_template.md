## Summary

Describe the user-visible behavior and any security boundary changed by this pull request.

## Access mode

- [ ] Bounded mode
- [ ] Operator mode
- [ ] Both / shared infrastructure
- [ ] Documentation or tooling only

## Security checklist

- [ ] Target authorization remains exact and fail-closed.
- [ ] Bounded-mode caller input does not become executable shell text.
- [ ] Any Operator-mode shell execution remains behind local opt-in and lease enforcement.
- [ ] TetherBound does not silently add root/sudo privileges.
- [ ] Remote content is treated as untrusted model input.
- [ ] Credentials, raw commands, and remote output are not logged by default.
- [ ] Time, request, and output bounds remain enforced where applicable.
- [ ] Tests cover allowed and rejected/expired paths.
- [ ] Documentation and threat-model assumptions are updated.
- [ ] No real credentials, private infrastructure identifiers, or secret-bearing logs are included.
