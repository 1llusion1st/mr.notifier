# mr.notifier

info:
=====
this app listens for gitlab merge requests open event and notifies in telegram channel every developer joined to the repository

base functionality:
===================
- configuration list of projects
- configuration list of reviewers
- editing reviewers list for repository using admin(authorized) chat(telegram)

build:
```bash
docker build -t notifier .
```

help:
```bash
docker run --rm notifier -h
Usage: mr.notifier <command>

Flags:
  -h, --help    Show context-sensitive help.

Commands:
  generate <config-file> <telegram.bot-api> <telegram.channel-id> <telegram.admin-id> <telegram.thread-id> <web-hook-path> <webhook-port> <new-projects> ...

  run <config-file>

Run "mr.notifier <command> --help" for more information on a command.
```

generate config example:
```bash
docker run --rm -it notifier  generate - BOT-ID 123123123 23423432 2 /webhook 7777 http://example.com/gitlabhq/gitlab-test,@user2,@user3 http://some-project/user/repo,@user3,@user4,@user5 
```

```yaml
telegram:
    bot-api: BOT-ID
    channel-chat-id: -123123123
    admin-chat-id: 23423432
    thread-id: 2
projects:
  - project: http://example.com/gitlabhq/gitlab-test
    reviewers:
      - '@user2'
      - '@user3'
  - project: http://some-project/user/repo
    reviewers:
      - '@user3'
      - '@user4'
      - '@user5'
reviewers:
  - '@user3'
  - '@user4'
  - '@user5'
  - '@user2'
web-hook-path: /webhook
web-hook-port: 7777
```

run:
====
```bash
docker run --rm -v /tmp/config.yaml:/config.yaml notifier run /config.yaml
```

bugs:
-----
- kong framework - can't pass negative chat id number as an argument - at this moment app changes positive number to negative.