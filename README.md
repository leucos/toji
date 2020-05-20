# Toji

[![License: WTFPL](https://img.shields.io/badge/License-WTFPL-brightgreen.svg)](http://www.wtfpl.net/about/)
[![pipeline status](https://gitlab.com/leucos/toji/badges/master/pipeline.svg)](https://gitlab.com/leucos/toji/-/commits/master)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/leucos/toji)](https://goreportcard.com/report/gitlab.com/leucos/toji)

Toji is a Toggle ➡ Jira bridge. Make time tracking ~fun again~ less painful.

The active project is on [Gitlab](https://gitlab.com/leucos/toji/).

## Objectives

Toji takes your Toggl time entries and adds them in Jira issues worklog. No
need to do it manually anymore.

If you use the [Toggl
Button](https://chrome.google.com/webstore/detail/toggl-button-productivity/oejgccbfbmkkpaidnkphaiaecficdnfn)
brower extension and configure Jira integration, your Toggl tasks will start
with the Jira issue. Backporting Toggl time entries to Jia will be a breeze
using Toji.

## Install

Go to [releases pages](https://gitlab.com/leucos/toji/-/releases), download the
binary for your architecture.

You can also [build](#building) it yourself.

## Usage

Get your [Toggl API Token](https://toggl.com/app/profile) and
[Atlassian API](https://id.atlassian.com/manage-profile/security) tokens handy,
and create a configuration calling:

```bash
toji init
```

the configuration will be saved in `$XDG_CONFIG_HOME/toji/config.yml` (usually
`~/.config/toji/config.yml`).

You can then import your Toggl today's entries using:

```bash
toji sync today
```

Example dry-run

```
$ ./bin/toji sync thu -t fri -n

Syncing toggl entries between 2020-04-23 00:00:00 +0200 CEST and 2020-04-24 23:59:59 +0200 CEST

Thu 2020/04/23
==============

        DEV-399 Develop a Toggl->Jira bridge
                [07:21 - 08:23] would insert 1h 2m 12s from entry 1125172937 to DEV-399's worklog entry
                [08:53 - 10:10] would insert 1h 16m 43s from entry 1125305414 to DEV-399's worklog entry

        DEV-419 Helm Charts elasticsearch
                [10:10 - 10:43] would insert 0h 33m 22s from entry 1125410066 to DEV-419's worklog entry

        DEV-399 Develop a Toggl->Jira bridge
                [12:42 - 13:33] would insert 0h 51m 43s from entry 1125616608 to DEV-399's worklog entry

        DEV-377 Meetings: Team meeting
                [13:33 - 14:02] would insert 0h 28m 21s from entry 1125706034 to DEV-377's worklog entry

        DEV-399 Develop a Toggl->Jira bridge
                [14:02 - 15:46] would insert 1h 44m 8s from entry 1125756948 to DEV-399's worklog entry

Fri 2020/04/24
==============

        DEV-399 Develop a Toggl->Jira bridge
                [06:09 - 10:20] would insert 4h 11m 12s from entry 1524637903 to DEV-399's worklog entry

        DEV-377 Meetings: Team meeting
                [13:00 - 14:34] would insert 1h 33m 33s from entry 1525159511 to DEV-377's worklog entry

        DEV-422 Keycloak provisionning via Terraform
                worklog entry DEV-422 for 2h 24m already exists
```

See [detailed usage](#detailed-usage) for more information.

## Building

```bash
go build .
```

A `Makefile` is also provided if you wish with the following targets:

| target           | description                                       |
| ---------------- | ------------------------------------------------- |
| all              | Build binary for your arch (default)              |
| linux            | Build linux binary                                |
| darwin           | Build MacOS binary                                |
| windows          | Build Windows binary                              |
| release          | Build programall binaries for release             |
| test-bench       | Run benchmarks                                    |
| test-short       | Run only short tests                              |
| test-verbose     | Run tests in verbose mode with coverage reporting |
| test-race        | Run tests with race detector                      |
| check test tests | Run tests                                         |
| test-xml         | Run tests with xUnit output                       |
| test-coverage    | Run coverage tests                                |
| lint             | Run golint                                        |
| fmt              | Run gofmt on all source files                     |
| clean            | Cleanup everything                                |

## Configuration

See [Init](#init).

## Detailed usage

You can see the complete usage executing:

```bash
toji -h
```

The more interesting parts are highlighted below.

### Completion

To install bash completion, just run:

```
. <(toji completion)
```

If you want completion loaded when your shell start, add the above line in your
`~/.bashrc`.

### Init

`init` will create or update your existing configuration.

You will need your
[Jira API token](https://id.atlassian.com/manage-profile/security) and your
[Toggl API Token](https://toggl.com/app/profile).

```bash
toji init
```

The config file will be written in `$XDG_CONFIG_HOME/toji/config.yml`[^1] by
default. If `$XDG_CONFIG_HOME` is not set, `~/.config/toji/config.yml` will be
used instead.

If you want to use another path, you can invoke `toji` with `-c` or `--config`.

Configuration have default values and specific profiles:

```yaml
jira:
  token: abcd1234
  url: https://corp.atlassian.net
  username: me@corp.com
toggle:
  token: z81c1760588de3cd8d4fb43479973039
profiles:
  anothercorp:
    jira:
      token: egfh5678
      url: http://anothercorp.atlassian.net
      username: me@anothercorp.com
    toggle:
      token: zf8f15509a99d75879e03450a74d19b1
```

The profile `anothercorp` can now be sledcted using `-p` (or `--profile`) when
invoking `toji`.

If a specific key is not present in the profile (for instance, no `toggl.token`
in `anothercorp` profile), the default will be used. This way, you can have a
single Toggl account, and specific profiles with differents Jira profiles.

To add a specific profile, invoke `toji` as follows:

```bash
toji init -p anothercorp
```

If a profile (of default) already exists, `toji` will refuse to overwrite it.

### Sync

Sync will fetch Toggl time entries in the requested period and add them to
corresponding Jira issues. Toji will try to match the issue key in the
beginning of the Toggl entry description.

For instance, if a Toggl entry has the description
`DEV-123 Create a toggl -> jira bridge`, Toji will match `DEV-123` as the issue
 key and try to update it's time entries.

Time entries are added in the Jira worklog for the issue. Toji will insert a
special value (`toggl_id`) in this field so it does not try to insert the same
Toggl time entry twice in an issue. In this respect, Toji is idempotent.

The message added to the worklog can be asked interactively when `-i` is used.
If not, the Toggl entry description (minut the issue key) will automatically be
used.

The general command is:

```
toji sync <start> [--to <end>] [--dryrun] [--only issue1,issue2] [--interactive]
```

where:

- `start` and `end` are respectively the starting and ending date for the sync
  operation (more on this below)
- `--dryrun` show which entris would be added in Jira issues (alias: `-n`)
- `--only` is the list of the only Jira issues we want to update (alias: `-o`)
- `--interactive` will make Toji ask for a single line comment for enach added
  entry; this comment will be added to the worklog (alias: `-i`)

When specifiying `--only`, you can supply a comma-separated list of issue, or
repeat the option several times (or both).

For instance, the command `toji sync today -o DEV-22,DEV-45 -o DEV-55` will
only update time tracking entries in issues DEV-22, DEV-45 and DEV-55.

The `--interactive` flag will ask for a comment fort each new entry.
See [comments](#comments) for description.

Start and end times can be specified in a variety of ways.

- `today`: entries from today
- `yesterday`: entries from yesteray
- `week`: this week
- `month`: this month
- `year`: this year
- `monday` (or any other day of the week): last monday
- `YYYYMMDD[HHMM]`: absolute date

If the end date (`--to`) is not specified, the same value as `start` will be
used (e.g. `toji sync yesterday` is equivalent to `toji sync yesterday -to
yesterday`)

This is quite obvious but has to be said: the `start` date must precede the
`end` date. The end date, however, can be in the future (day of week are only
considered in the past though).

Note that the time range considered always starts at 00h00 for the start value,
and 23h59 for the end value.

When symbolic dates (e.g. `week`, `yesterday`, ...) are used, the generated
date depends on context. The start date is 00:00 in the first day of the
period. if used in to, it will be the 23:59 in the last day of the period. if
-to is omitted, the first value is implied (e.g. `toji sync yesterday` is
equivalent to `toji sync yesterday -to yesterday`).

Finally, weekdays have short form equivalents (`mon`, `tue`, `wed`, `thu`,
`fri`, `sat`, `sun`) for convenience.

#### Comments

When `--interactive` (or `-i` is used), Toji will ask for a comment on each new
entry, otherwise the Toggl description (minus `[ISSUE-ID]`) is used.

If you do not wish to enter a comment, just press enter.

Comments are multiline and an empty line will validate the comment.

As a bonus, if you start your comment with `*`, this comment will also be added
to the Jira issue comments (and the worklog as usual).

#### Examples

##### Sync all Toggl entries between yesterday at 00h00 and yesterday at 23h59

```bash
toji sync yesterday
```

##### Show which entries would be added to Jira worklog but not add them (dry run mode)

```bash
toji sync yesterday -n -i
```

Also asks a comment interactively for each added entry.

##### Sync all Toggl entries found for issues DEV-55 and DEV-69 between yesterday at 00h00 and yesterday at 23h59

```bash
toji sync yesterday -o DEV-55,DEV-69
```

##### Sync all Toggl entries between monday 00h00 of the current week and sunday 23h59 of the current week

```bash
toji sync week
```

##### Sync all Toggl entries between last tuesday at 00h00 and last thursday at 23h59

You can not execute this on tuesdays, wednesdays or thursdays of course since
`end` would be before `start`.

```bash
toji sync tuesday --to thursday
```

##### Same as above

```
toji sync tue --to thu
```

##### Sync all Toggl entries between march 12th 2020 at noon and last tuesday at 23h59

```
toji sync 202004121200 --to tue
```

##### Sync all Toggl entries between june 2nd 2020 at june 21st 2020 (entire days)

```
toji sync 20200602 --to 20200621
```

## Caveats

Toji is in GoodEnough™ (i.e. "works on my machine") state.

Use at your own risks (a.k.a. "no tests").

Don't [drink bleach](https://twitter.com/RandyRainbow/status/1254062239595859975).

## Thanks

Thanks [@devatoria](https://github.com/Devatoria) for cobra layout inspiration
& [@earzur](https://github.com/earzur) for β-testing.

## Licence

WTFPL

Contribs welcome.

## References

[^1]: https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html