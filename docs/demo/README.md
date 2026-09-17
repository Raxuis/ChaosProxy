# Demo recording

`docs/assets/demo.gif` shows Chaos Proxy in front of
[RealWorld Conduit](https://github.com/gothinkster/realworld): the
[Vue 3 frontend](https://github.com/gothinkster/vue-realworld-example-app) and the
[Go/Gin API](https://github.com/gothinkster/golang-gin-realworld-example-app).
Regenerate it after a visible dashboard change.

Requirements: Go, Node.js 20+, Google Chrome, ImageMagick, and gifski.

1. Start the API on port 9000, then seed two users and four articles from the
   repository root:

   ```sh
   git clone https://github.com/gothinkster/golang-gin-realworld-example-app
   cd golang-gin-realworld-example-app
   PORT=9000 go run .
   ```

   ```sh
   docs/demo/seed.sh
   ```

2. Start Chaos Proxy with the demo configuration from the repository root:

   ```sh
   go run ./cmd/chaosproxy --config docs/demo/chaos.yaml
   ```

3. Start the frontend on port 3001 against the proxy:

   ```sh
   git clone --recurse-submodules https://github.com/gothinkster/vue-realworld-example-app
   cd vue-realworld-example-app
   npm install
   VITE_API_URL=http://localhost:7070/api npx vite --port 3001
   ```

4. Record the frames with headless Chrome, then compose the GIF:

   ```sh
   npm install --no-save playwright-core
   node docs/demo/record.mjs /tmp/chaosproxy-frames
   docs/demo/compose.sh /tmp/chaosproxy-frames
   ```

`record.mjs` prints the page errors it saw; the demo expects
`Cannot read properties of null (reading 'slug')`.
