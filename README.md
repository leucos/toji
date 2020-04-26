# Toji

[![License: WTFPL](https://img.shields.io/badge/License-WTFPL-brightgreen.svg)](http://www.wtfpl.net/about/)
[![pipeline status](https://gitlab.com/leucos/toji/badges/master/pipeline.svg)](https://gitlab.com/leucos/toji/-/commits/master)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/leucos/toji)](https://goreportcard.com/report/gitlab.com/leucos/toji)

Toji is a Toggle ➡ Jira bridge. Make time ~tracking fun again~ less painful.

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

Get your Toggl and Jira API tokens handy, and create a configuration calling:

```bash
toji init
```

the configuration will be saved in `$XDG_CONFIG_HOME/toji/config.yml` (usually
`~/.config/toji/config.yml`).

You can then import your Toggl today's entries using:

```bash
toji sync today
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

For instance, if a Toggl entry has the description `DEV-123 Create a toggl ->
jira bridge`, Toji will match `DEV-123` as the issue key and try to update it's
time entries.

Time entries are added in the Jira worklog for the issue. Toji will insert a
special value (`toggl_id`) in this field so it does not try to insert the same
Toggl time entry twice in an issue. In this respect, Toji is idempotent.

The general command is:

```
toji sync <start> [--to <end>] [--dryrun] [--only issue1,issue2]
```

where:

- `start` and `end` are respectively the starting and ending date for the sync
  operation (more on this below)
- `--dryrun` show which entris would be added in Jira issues (alias: `-n`)
- `--only` is the list of the only Jira issues we want to update (alias: `-o`)

When specifiying `--only`, you can supply a comma-separated list of issue, or repeat the option several times (or both).

For instance, the command `toji sync today -o DEV-22,DEV-45 -o DEV-55` will
only update time tracking entries in issues DEV-22, DEV-45 and DEV-55.

Start and end times can be specified in a variety of ways.

- `today`: entries from today
- `yesterday`: entries from yesteray
- `week`: this week
- `month`: this month
- `year`: this year
- `monday` (or any other day of the week): last monday
- `YYYYMMDDHHMM`: absolute date

If the end date (`--to`) is not specified, the same value as `start` will be
used (e.g. `toji sync yesterday` is equivalent to
`toji sync yesterday -to yesterday`)

This is quite obvious but has to be said: the `start` date must precede the
`end` date. The end date, however, can be in the future (day of week are only
considered in the past though).

Note that the time range considered always starts at 00h00 for the start value,
and 23h59 for the end value.

Finally, weekdays have short form equivalents (`mon`, `tue`, `wed`, `thu`,
`fri`, `sat`, `sun`) for convenience

#### Example

`toji sync yesterday` will sync all Toggl entries between yesterday at 00h00
  and yesterday at 23h59

`toji sync yesterday -n` will show which entries would be added to Jira worklog but not add them (dry run mode)

`toji sync yesterday -o DEV-55,DEV-69` will sync all Toggl entries found for
issues DEV-55 and DEV-69 between yesterday at 00h00 and yesterday at 23h59

`toji sync week` will sync all Toggl entries between monday 00h00 of the
  current week and sunday 23h59 of the current week

`toji sync tuesday --to thursday` will sync all Toggl entries between last
  tuesday at 00h00 and last thursday at 23h59 (you can not invoke this on tue,
  wednesday or thursday of course since `end` would be before `start`)

`toji sync tue --to thu` does the same as above

`toji sync 202004121200 --to tue` will sync all Toggl entries between last
  march 12th 2020 at noon and last tuesday at 23h59

when symbolic datespecs are used, the generated date depends on context.
the start date is 00:00 in the first day of the period.
if used in to, it will be the 23:59 in the last day of the period.
if -to is omitted, the first value is implied
(e.g. `toji sync yesterday` is equivalent to `toji sync yesterday -to yesterday`)

## Caveats

Toji is in GoodEnough™ (i.e. "works on my machine") state.

Use at your own risks (a.k.a. "no tests").

Don't [drink bleach](https://twitter.com/RandyRainbow/status/1254062239595859975).

## Licence

WTFPL

Contribs welcome.

## References

[^1]: https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html