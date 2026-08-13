# kageos License FAQ

This FAQ explains the project license in plain language. It is not legal advice,
and the full [LICENSE](LICENSE) controls if anything here is unclear or
inconsistent.

## What license does kageos use?

The kageos core platform is licensed under the Business Source License 1.1
(`BSL 1.1`) with Apache License 2.0 as the change license.

kageos is source-available today. It is not OSI open source today because the
current license restricts unauthorized commercial products, services,
distributions, and offerings built from the kageos core platform.

## Can I self-host kageos?

Yes. You can self-host kageos for yourself, your team, or your organization.

## Can my company use kageos internally?

Yes. A company can deploy and use kageos for its own internal work, including
production use, as long as it is not offering kageos as a Restricted Commercial
Offering to third parties.

The BSL grant carves out access by your employees and contractors. If your use
exposes kageos to external customers as part of a commercial product, hosted
service, managed service, white-label product, OEM product, embedded product,
or competing offering, contact the maintainers for commercial authorization.

## Does self-hosting have to be on an internal network?

No. The license does not require kageos to run only on a private LAN.

You may deploy kageos on your own server, cloud account, VPN, intranet, or a
publicly reachable domain, as long as the instance is for you, your team, your
organization, employees, or contractors, and you are not offering kageos as a
Restricted Commercial Offering to third-party users.

For example:

- A personal instance reachable from the internet is allowed.
- A company instance on AWS, Azure, GCP, or another cloud is allowed when used
  by that company's employees and contractors.
- A private enterprise instance exposed through SSO, VPN, or a public domain is
  allowed when access is limited to that organization and its contractors.
- Selling kageos access, hosted workspaces, managed service access, white-label
  packages, OEM packages, embedded products, or competing kageos-like products
  to external customers is not allowed under the default BSL grant and requires
  commercial authorization.

Consulting, implementation, and support are different from restricted
commercial offerings. A consultant may help a customer deploy, configure,
integrate, train on, or support kageos in the customer's own cloud account,
servers, or private environment, where the customer controls the instance and
the users are that customer's employees or contractors.

A consultant, agency, or managed-service provider may not host kageos as its
own commercial managed service for one or more external customers under the
default BSL grant. That requires commercial authorization from the kageos
maintainers.

A vendor may not package kageos as a white-label, OEM, embedded, rebranded, or
on-premises commercial product for third-party production use under the default
BSL grant. That also requires commercial authorization.

## Can I modify the code?

Yes. You can inspect, modify, and create derivative works from the code under
the terms of the BSL 1.1 license.

## Can I redistribute my modified version?

Yes, under the BSL 1.1 terms. Modified copies and derivative works remain under
the same license until the applicable version reaches its Change Date. Keep the
license notice with the code and do not add rights that the BSL does not grant.

Redistribution does not let you bypass the Restricted Commercial Offering
limits. Commercially distributing kageos, or a modified version of kageos, as a
white-label, OEM, embedded, rebranded, on-premises, managed, hosted, or
competing product for third-party production use requires commercial
authorization.

## Can I offer kageos as a commercial SaaS, hosted service, or product?

Not under the default BSL grant.

You may not use the kageos core platform to provide a Restricted Commercial
Offering to third parties. That includes commercial SaaS, hosted services,
managed services, white-label products, OEM products, embedded products,
rebranded products, on-premises commercial packages, resale, and competing
products or services whose main value substantially overlaps with kageos.

Those uses require commercial authorization from the kageos maintainers.

## Which commercial uses require authorization?

The following uses are not allowed under the default BSL grant and require
commercial authorization from the kageos maintainers:

- Offering kageos as a commercial SaaS, hosted service, paid online workspace,
  or managed service for external users.
- Operating kageos as a managed service provider (`MSP`), agency, or consulting
  vendor where you host and operate kageos for external customers as your own
  paid service.
- Reselling hosted kageos access, seats, accounts, workspaces, or tenant
  instances to third parties.
- Offering a white-label, OEM, embedded, rebranded, or on-premises commercial
  product where kageos is a core value-driving component provided to external
  customers for production use.
- Modifying, forking, rebranding, or removing kageos marks from the code and
  then selling substantially similar hosted kageos functionality to third
  parties.
- Packaging kageos into a competing product or service whose main value
  substantially overlaps with the kageos core platform.

Changing the name, user interface, deployment scripts, or branding does not
remove the BSL restrictions. Modified copies and derivative works remain under
the BSL 1.1 terms until the applicable version reaches its Change Date.

If you want to provide any of the uses above, contact the maintainers for a
commercial license before offering the service.

## When does a version convert to Apache-2.0?

Each released version converts to Apache License 2.0 four years from that
version's release date, unless that version states a different Change Date.

After a specific version reaches its Change Date, that version is available
under Apache-2.0. Newer versions may still be under BSL 1.1 until their own
Change Dates.

The maintainers may choose an earlier Change Date for a future version or
specific release if that is better for the project and community.

## Are the SDK, examples, and templates also BSL?

Unless a file, package, or directory has its own license notice, material in
this repository follows the root [LICENSE](LICENSE).

That means SDK reference material, templates, seed examples, and example
capability bundles in this repository are BSL 1.1 by default today and convert
with the repository version after the Change Date.

The project may publish some SDKs, examples, templates, or docs under separate
permissive licenses such as Apache-2.0 or MIT. When that happens, the license
file or header in that specific package controls.

## Can I use the kageos name or logo?

Not automatically. The BSL license does not grant trademark or logo rights.
Ask the maintainers before using kageos branding in a product, service, or
public distribution.

## What should I call this project publicly?

Use:

- "source-available"
- "self-hostable"
- "public source code"
- "BSL 1.1, converts to Apache-2.0 after four years"

Avoid calling the current BSL-licensed core "OSI open source" or "fully open
source". It is not.
