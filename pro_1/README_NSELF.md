# Nhost Self-hosted Project

This project is a self-hosted instance of Nhost, providing GraphQL APIs, authentication, storage, and more.

## Getting Started

1. **Initialize the project:**
   ```bash
   nself init
   ```

2. **Configure environment variables:**
   Edit the `.env.dev` file with your settings, ensuring the `HOSTS` variable is correctly set. For example:
   ```env
   HOSTS=dashboard.nproj.run,graphql.nproj.run,auth.nproj.run
   # Add more hosts as needed, separated by commas
   ```

3. **Start the services:**
   ```bash
   nself up
   ```

4. ** Stop the service:**
   ```bash
   nself down
   ```
## Project Structure

- `.env.dev or .env`: Environment configuration file.
- `emails/`: Email templates.
- `functions/`: Serverless functions.
- `services/`: User-created services and scripts.
- `docker-compose.yml`: Docker Compose configuration file.

## Docker Compose

The `docker-compose.yml` file is generated based on the environment settings and starts all necessary services.

## Additional Resources

- [Nhost Documentation](https://docs.nhost.io/)
- [Hasura Documentation](https://hasura.io/docs/)
