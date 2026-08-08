## Summary

Describe the behavior and security boundary changed by this pull request.

## Security checklist

- [ ] No MCP input becomes arbitrary shell text.
- [ ] Target and service authorization remains fail-closed.
- [ ] Remote content is treated and labeled as untrusted data.
- [ ] Secrets, commands, and remote output are not logged.
- [ ] Time and output bounds remain enforced.
- [ ] Tests cover the new behavior and its rejection paths.
- [ ] Documentation and threat-model assumptions are updated.
