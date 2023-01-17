## Env variable instructions

Environment types (`${environment}`): `dev` and `production`
For local development use `dev`


- `.env.${environment}` - will be used for general enironmental variables which are not secrets
- `.env.${environment}.local` will be used for secret variables only