# TetherBound validation site

This directory is a dependency-free static site for product validation. It is
intentionally separate from the gateway runtime and does not require Node, a
framework, analytics, or a backend.

## Preview locally

From the repository root:

```bash
python3 -m http.server 4173 --directory site
```

Then open:

```text
http://127.0.0.1:4173/
```

Do not open the HTML files directly with `file://`; serving them locally catches
relative-link and hosting-path issues more realistically.

## Pages

- `index.html` — validation landing page.
- `security.html` — customer-facing trust and security model.
- `community.html` — current open-source / DIY edition.
- `cloud.html` — proposed hosted edition and early-access CTA.
- `docs.html` — documentation portal linking the implemented and proposed specs.
- `styles.css` — shared visual system; no external fonts or tracking resources.

## Validation CTA

The Cloud CTA opens the structured GitHub issue form at:

```text
.github/ISSUE_TEMPLATE/cloud-interest.yml
```

The form intentionally asks for product context rather than infrastructure
details and warns users not to submit credentials, private addresses, logs,
customer data, or other secrets.

Before a broader public launch, consider moving interest collection to a proper
form service or first-party backend with a published privacy notice. Do not add
visitor fingerprinting or session replay as part of early security-product
validation.

## Deployment

The site can be hosted from this directory on GitHub Pages, Cloudflare Pages,
Netlify, S3/static hosting, or any ordinary web server. There is no build step.

A production deployment should add only after a domain is selected:

- canonical URLs;
- Open Graph/social image;
- favicon/app icons;
- domain-specific privacy and terms links;
- a real early-access form endpoint;
- minimal, documented analytics if product-learning requires it.

Do not add a deployment workflow that automatically publishes this private
validation branch without an explicit product decision.

## Claims discipline

The site uses three visibly different states:

- **Available / Working** — implemented and tested.
- **Planned / Roadmap** — committed design but not shipped.
- **Validation / In development** — hosted-product hypothesis.

Keep those distinctions intact. See `docs/product-positioning.md` for the full
messaging and claims guide.
