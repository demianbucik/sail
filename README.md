# Sail
Sail is a simple form backend API for sending emails from HTML and JavaScript forms. With Google Cloud's and SendGrid's free tier, it can verify and send 12000 emails each month completely cost free.

Deployment process is really simple and only takes a few minutes.

To avoid spam sent by bots, you can choose between reCAPTCHA versions 2 and 3. YAML configuration file enables the setup of custom redirects and confirmation email templates that support macros.

## Configuration and deployment
### Google Cloud and SendGrid
In case you don't have an account, you register and create a project here: [https://console.cloud.google.com/](https://console.cloud.google.com/).
Sign up for SendGrid's free plan here: [https://console.cloud.google.com/marketplace/product/sendgrid-app/sendgrid-email](https://console.cloud.google.com/marketplace/product/sendgrid-app/sendgrid-email). Free tier includes 12000 monthly emails.
You also have to verify your domain ([https://app.sendgrid.com/guide](https://app.sendgrid.com/guide)) and obtain an API key.

### Configuration
Create a new configuration file from the provided example file with `cp example.env.yaml env.yaml`.
Provide your SendGrid and reCAPTCHA secret keys, configure your and noreply email addresses and names, redirect pages, and confirmation email template.

Supported macros in confirmation email:
- {{ .FORM_NAME }}
- {{ .FORM_EMAIL }}
- {{ .FORM_SUBJECT }}
- {{ .FORM_MESSAGE }}
- {{ .NOREPLY_NAME }}
- {{ .NOREPLY_EMAIL }}
- {{ .RECIPIENT_NAME }}
- {{ .RECIPIENT_EMAIL }}

To disable reCAPTCHA verification, leave the version field empty (secret key will be ignored). All other environment variables are required.

The form message and confirmation emails will have _reply-to_ fields configured to the other persons actual email address.

### Deployment
You can either deploy the function by executing the deployment script `./deploy.sh`, which requires `gcloud` command-line tool ([https://cloud.google.com/sdk/docs/install](https://cloud.google.com/sdk/docs/install)).
Or upload the zipped content of this repo directly via the web console ([https://console.cloud.google.com/functions/list](https://console.cloud.google.com/functions/list)).

## Local testing and development
Check the provided example, it provides a simple form, redirect pages and a local server.
Local server handles form submissions and serves the HTML pages. That way you can verify your setup works before deployment.

## Contributing
Pull requests as well as feature requests are welcome. For major changes, please open an issue first to discuss.