# Dr-Docso

A Go documentation bot for Discord, written with
[hhhapz/doc](https://github.com/hhhapz/doc).

---

Dr-Docso is written in Go, and uses slash commands to query Go documentation
with a single command (`/docs query:strings.split`).

The bot will display minimal content, and allow users to view expanded
content privately, to prevent hindrance to other users and the conversation
topic.

The bot is created to be used in the [Discord Gophers](https://discord.gg/golang).

## Usage

```discord
/docs query:fmt
/docs query:fmt.Errorf
/docs query:fmt Errorf
/docs query:github.com/hhhapz/doc.package
/docs query:github.com/hhhapz/doc searcher search
/docs query:http
/docs query:net/http
```
