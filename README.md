
# nself

`nself` is an open-source CLI tool for managing a self-hosted Nhost backend stack using Docker Compose. It allows you to initialize, start, stop, reset, and update your self-hosted Nhost services with ease.

## Features

- **Init**: Initialize your project with a customizable `.env` file.
- **Up**: Start all enabled Docker services based on your configuration.
- **Down**: Stop all running Docker services.
- **Reset**: Reset your environment by removing Docker containers, images, volumes, and the Docker Compose file.
- **Update**: Update `nself` to the latest version.

## Installation

You can install `nself` using the provided `install.sh` script.

```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

**Note:** Ensure you have `curl` and `bash` installed on your system.

## Usage

After installation, navigate to your project directory and use the following commands:

### Initialize the Project

```bash
nself init
```

This will copy the `.env.example` to `.env.dev`. Modify the `.env.dev` file according to your needs and then run:

```bash
nself init
```

to generate the `.env` file.

### Start Services

```bash
nself up
```

This command will generate the `docker-compose.yml` based on your `.env` configuration and start all enabled services.

### Stop Services

```bash
nself down
```

This will stop all running Docker services defined in your `docker-compose.yml`.

### Reset Environment

```bash
nself reset
```

This command stops services, removes Docker images and volumes, and deletes the `docker-compose.yml` file, allowing you to reconfigure and rebuild your environment.

### Update nself

```bash
nself update
```

Checks for the latest version of `nself` and updates your local CLI if a newer version is available.

## Configuration

Edit the `.env.dev` or `.env` file to configure your services. The `.env.example` provides all necessary environment variables with explanations.

## License

MIT License. See [LICENSE](LICENSE) for more information.
