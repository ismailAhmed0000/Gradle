import { defineRailway, github, postgres, redis, service, project } from "railway/iac";

export default defineRailway(() => {
  const db = postgres("postgres");
  const cache = redis("redis");

  const api = service("go-backend", {
    source: github("ismailAhmed0000/Gradle", {
      branch: "main",
      rootDirectory: "go-backend",
    }),
    env: {
      DATABASE_URL: db.env.DATABASE_URL,
      REDIS_URL: cache.env.REDIS_URL,
    },
  });

  return project("gradle-backend", {
    resources: [db, cache, api],
  });
});
