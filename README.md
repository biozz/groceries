# Groceries

Me and my wife never really liked any of the todo apps out there when it came to regular shopping. They were either too complicated or too limited. So I decided I can do just the right thing for us. This is how Groceries appeared.

Main features include:

- web app only (for now)
- unlimited number of tasks (hi, Todoist)
- free and the only costs are my self managed server and the domain (hi \<any todo app on the AppStore\>)
- dead-simple storage setup (you can use any redis-compatible solution, like Redis itself or Bitcask)
- dead-simple interface, even my mom can use it (hi, Taskwarrior)
- does not want to be a note taking app or something more than a todo list (hi, Notion, Joplin, NextCloud, etc.)
- does not pretend to solve your todo needs, your milage may vary, it just works for my family
- painless grouping by category, no messing around with sub tasks (hi, Reminders)
- predictable live updates via WebSockets (oh, hi again, Reminders)
- just a single binary for any server platform with embedded static files (thank you, Go)

## Usage

Binary:

```bash
./groceries -bind=:8080 -kvhost=localhost:6379
```

Docker (after starting redis):

```bash
docker run -d groceries:latest -bind=:8080 -kvhost=redis:6379
```

## License

MIT
