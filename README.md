##  Notifier Bot

This is a bot written in Go that sends links of games played by members of nyumbani mates team on lichess to telegram .
Its a remake of [community bot](https://github.com/swahili-chess/community-bot) which i previous wrote in Javascript.
As we all know Javascript comes with weird unknown and unusual errors that leads to bot shutting down, hence it was necessary to rewrite
with a more capable language for the task (Go).

## How it works

The bot monitors games played by members of nyumbani mates team on lichess and extracts the links of these games. It then sends these links to a specified Telegram Chat with 
[@chesswahiliBot](https://t.me/chesswahiliBot).



## Contributing

If you would like to contribute . please follow these steps:

1. Fork this repository.
2. Create a new branch: `git checkout -b my-new-feature`.
3. Make your changes and commit them: `git commit -am 'Add some feature'`.
4. Push to the branch: `git push origin my-new-feature`.
5. Submit a pull request.

## License

This project is licensed under the MIT License.
