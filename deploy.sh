#!/bin/bash

NAME=$1

gcloud functions \
    deploy \
    "$NAME" \
    --trigger-http \
    --allow-unauthenticated \
    --source=. \
    --region=europe-west3 \
    --entry-point=SendEmailHandler \
    --memory=128MB \
    --runtime=go116 \
    --timeout=5 \
    --max-instances=2 \
    --env-vars-file=./env.yaml
