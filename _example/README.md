# Example

## Form setup

This example will help you set up and verify your email form and Sail configuration file.
File `form.html` contains a basic form. Depending on whether the form submission was successful or not, the server will redirect to `thankyou.html` or `error.html`.

All required setting can be configured by modifying the `env.yaml` file, which is used for configuring both the local and production environment.

The `server.go` file contains a simple local development server.
It handles form submissions and well as serving the static HTML files.
To start the server, you can just run one of the precompiled binaries, for example:
```bash
./server-linux-amd64
```
You will be able to download them with the first release.
Use the `-env` flag to specify a custom environment file, otherwise it will default to `../env.yaml`.
Use `-help` for more information.

After you run start the server, visit `http://localhost:8000` with your browser and open the `form.html` file. Fill in the fields and try to submit it. 

If you have a Go compiler installed locally, you can of course run the server with `go run server.go`.




