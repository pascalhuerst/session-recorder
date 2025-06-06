# Session Recorder Web

## Development

### Libraries

#### Session Waveform (Player App)

The easiest way to work on the Player app is by launching Storybook:

```
npx nx storybook --project session-waveform
```

### Application

#### Dev Mode

To run application in dev mode, go to `/web` directory:

##### Run `npm install`

##### Start proxy for grpc web

Run envoy in docker container: (You might have to adapt the config)

```
cd grpc-web-proxy
docker-compose up envoy
```

##### Setup .env

- create `.env` using `.env.example` as a reference

##### Start the app

- run npm `npm start`
