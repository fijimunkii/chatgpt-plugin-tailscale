# Security policy

Please do not open a public issue for a vulnerability that could expose a tailnet, credential, or remote host. Use GitHub's private vulnerability reporting feature for this repository when available.

Include:

- affected commit or release;
- deployment mode and relevant non-secret configuration;
- a minimal reproduction;
- expected and observed security boundary;
- impact assessment.

Do not include live auth keys, OAuth client secrets, bearer tokens, Tailscale state files, private host data, or command output containing secrets.

The project is pre-1.0. Only the latest commit on the default branch is intended to receive security fixes during this initial stage.
