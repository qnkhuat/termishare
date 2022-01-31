# To dev
Open 2 windows:
1. Run shadow-cljs: `npm run dev`
2. Run server and server the front-end: `clj -M:dev:cider-clj`
- ```
(use 'dev)
(dev/start!)
```

# To build
Just run `clj -T:build uber`. It will automatically:
- Install front-end dependencies
- Build front-end
- Compile back-end to uber jar

# Run
- TERMISHARE_PORT=3000 java -jar termishare.jar
