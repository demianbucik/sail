#!/bin/bash

NAME=$1

gcloud functions \
    deploy \
    "$NAME" \
    --gen2 \
    --trigger-http \
    --allow-unauthenticated \
    --source=. \
    --region=europe-west3 \
    --entry-point=SendEmailHandler \
    --memory=256MB \
    --runtime=go120 \
    --timeout=7 \
    --max-instances=2 \
    --env-vars-file=./env.yaml
#    --docker-registry=artifact-registry \
