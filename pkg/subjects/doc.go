// Package subjects is the single source of truth for cross-service NATS
// subjects.
//
// Naming convention:
//  1. Use dot-delimited tokens: <target>.<version>.<kind>.<domain>.<action>[.<scope...>].
//  2. target is the primary consumer/owner of the subject, for example runtime,
//     app, gateway, control.
//  3. version is currently fixed to v1; future protocol changes should add a new
//     version token.
//  4. kind is limited to cmd / query / event / reply:
//     - cmd: asks the target to execute a command or state change
//     - query: asks the target to answer a request-reply query
//     - event: broadcasts lifecycle or observable state
//     - reply: receives an asynchronous reply
//  5. Static tokens use lowercase kebab-case; dynamic scope stays at the tail and
//     currently mainly means {user}.{app}.{version}.
package subjects
