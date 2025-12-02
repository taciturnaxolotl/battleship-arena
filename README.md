# Battleship Arena

This is a service I made to allow students in my `cs-1210` class to benchmark their battleship programs against each other.

## I just want to get on the leaderboard; How?

First ssh into the battleship server and it will ask you a few questions to set up your account. Then scp your battleship file onto the server!

```bash
ssh battleship.dunkirk.sh
scp memory_functions_yourname.cpp battleship.dunkirk.sh
```

## Development

Built with Go using [Wish](https://github.com/charmbracelet/wish), [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lipgloss](https://github.com/charmbracelet/lipgloss).

```bash
# Build and run
make build
make run

# Generate SSH host key
make gen-key
```

See `AGENTS.md` for architecture details.

The main repo is [the tangled repo](https://tangled.org/dunkirk.sh/battleship-arena) and the github is just a mirror.

<p align="center">
	<img src="https://raw.githubusercontent.com/taciturnaxolotl/carriage/master/.github/images/line-break.svg" />
</p>

<p align="center">
	&copy 2025-present <a href="https://github.com/taciturnaxolotl">Kieran Klukas</a>
</p>

<p align="center">
	<a href="https://github.com/taciturnaxolotl/battleship-arena/blob/main/LICENSE.md"><img src="https://img.shields.io/static/v1.svg?style=for-the-badge&label=License&message=MIT&logoColor=d9e0ee&colorA=363a4f&colorB=b7bdf8"/></a>
</p>
