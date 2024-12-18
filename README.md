# Mini-Scan

## Getting Started
1. Copy `.env.example` to `.env` and modify to specify proper values for your environment.
2. Run `docker compose up -d` to spin up the PubSub emulator, configuration services, publisher, and database. The database uses the default postgres port, so you should change that if you have another PG instance running locally.
3. Run `task run` to spin up the subscriber.
4. Use any database tool you prefer to inspect and query the database.

## Notes
For this exercise, I made the assumption that despite the scan message schema being defined in this repository, it was not something that I had control over, so I implemented by own type that unmarshalled the message in a way that facilitated storage without too many extra calls.

I chose Postgres simply because of my familiarity with it and how quickly I could spin it up and configure it. The table is simple and relies on transaction isolation and the `ON CONFLICT DO UPDATE` pattern to deal with race conditions that could emerge from multiple service instances running at the same time. I used `jackc/pgx` for its `pgxpool` and because I didn't want to waste too much time dealing with abstractions through standard interfaces. I'm using postgres, so I'm using a postgres-specific lib.

No database migration tooling was configured to save time. When you spin up the containers within `docker-compose.yml`, the PG container will pull in the SQL files within `./db` and use them to initialize the database. I put the table creation there. No extra steps need to be taken to prepare the database for the service.

The GCP PubSub use here is pretty straightforward. I kept the default client configuration, which uses StreamingPull and receives messages on multiple goroutines. Pushing the configuration higher would likely end up showing some bottlenecks around the database connection, but it would take a good deal of scaling both the subscriber and the publisher to get there. I ran the publisher with a rate of 100 messages per second for testing, but didn't bother pushing it further.

The configuration is provided by the environment, though I would probably opt to fetch sensitive configuration / secrets from a secret manager or config manager hosted within the service boundary. As it stands, the environment configuration is not very sophisticated.

Speaking of tooling, I use [Taskfile](https://taskfile.dev/) here for building and running, which makes using `.env` files easier and prevents me from having to pull in yet another dependency. I figure a tool dependency was easier to deal with. If you don't have it, you can set it up with `go install github.com/go-task/task/v3/cmd/task@latest`. I love spending time on POSIX-compliant Makefiles but it's better for me to use simplier modern tools when doing exercises like this or I end up shaving just about every yak I come across. If you don't want to use Task, you can always build the `subscriber` service manually and run it after exporting all of the values within `.env`.

I also didn't configure any additional linting, checking or vetting, such as staticcheck, gosec, or golanglint-ci. Normally these would be used at dev time but also in pipelines on commit.

No automated testing was written for this simply due to time constraints. Unit tests could be written for certain functionality, such as the `UnmarshallJSON` implementation, but integration testing would be ideal to cover the interactions between the subscriber and the PubSub service, the subscriber and the database, and with multiple subscribers operating on the same database. Using a persistence layer interface would allow some degree of mocking for testing certain scenarios, but since the race condition handling is done by a postgres query written in our handler, we should test it against a database here. Lots of tools exist for that, including the spin-it-up-yourself-and-run-tests method, but I didn't spend the time on it, opting for a functional service and writing up notes.

