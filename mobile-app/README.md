# mobile-app

The React Native client for Gradle, used by **students** (teachers use a separate web
dashboard — see the root README). Log in, view assignments, scan and submit your own answer
sheet with the camera, and check graded results — all against `go-backend`'s REST API.

See the root [`README.md`](../README.md) for how this fits into the rest of the system.

## Stack

| Concern | Library | Notes |
|---|---|---|
| UI framework | React Native 0.86 + TypeScript | |
| Styling | [NativeWind](https://www.nativewind.dev) | Tailwind classes via `className`; a few dynamic/computed styles (colors, PDF layout) fall back to plain `StyleSheet`/inline `style` since NativeWind can't express runtime-computed values |
| Navigation | [React Navigation](https://reactnavigation.org) (native-stack) | only used for the Tasks flow — see Architecture below, most of the app doesn't go through a navigator at all |
| State management | React Context + hooks | **no Redux, no Zustand, no external state library** — see below |
| Persistence | `@react-native-async-storage/async-storage` | the JWT is the only thing persisted on-device |
| HTTP | axios | one shared instance (`src/api/client.ts`) with a request interceptor that attaches the bearer token |
| Camera | `react-native-image-picker` | drives the answer-sheet scan flow |
| PDF viewing | `react-native-pdf` | question papers and composited/graded documents |
| Icons | `react-native-svg` | hand-rolled inline icon components, no icon font/library |

### State management

There's no global store — three separate `React.Context` providers, each scoped to what it
actually owns, composed in `App.tsx`:

- **`AuthContext`** — plain `useState` (user, token, `isBootstrapping`). Persists the JWT to
  `AsyncStorage` under the key `auth_token` (`src/api/tokenStorage.ts`) and re-validates it
  against `GET /api/auth/me` on launch before rendering anything past `SplashScreen`.
- **`AssignmentsContext`** / **`SubmissionsContext`** — `useReducer`, each keeping a small
  in-memory cache keyed by id with an explicit `idle | loading | loaded | error` status per
  entry, so revisiting a screen doesn't refetch unless you pass `force: true`. `SubmissionsContext`
  additionally runs a `setTimeout`-based poll loop (`startPollingComposited`/`stopPollingComposited`)
  for a submission's composited-document status while it's `pending`/`generating`.

Nothing is persisted except the auth token — assignment/submission data is refetched fresh each
app launch.

## Requirements

- Node >= 22.11.0
- A working React Native environment for the platform(s) you're targeting (Xcode + CocoaPods
  for iOS, Android Studio/SDK for Android) — see the [React Native environment setup guide](https://reactnative.dev/docs/set-up-your-environment)
  if you haven't done this before.

## Setup

```sh
npm install

# iOS only, first run and after any native dependency change
bundle install
bundle exec pod install
```

## Configuring the API endpoint

`src/api/client.ts` has a **hardcoded** `API_BASE_URL`. It currently points at the deployed
Railway backend. For local development against a locally-running `go-backend`, change it to:

- `http://localhost:8080` on iOS simulator
- `http://10.0.2.2:8080` on Android emulator (this is the special alias Android emulators use
  for the host machine's `localhost`)
- your machine's LAN IP (e.g. `http://192.168.x.x:8080`) for a physical device on the same network

## Running

Start Metro (the JS bundler) and run the app in one of two ways:

```sh
npm start

# in another terminal
npm run android
npm run ios
```

This produces a debug build wired to the Metro dev server — it needs Metro running (and, for a
physical device, either a USB connection with port forwarding or the same Wi-Fi network) to load
the JS bundle.

To install a build that keeps working after disconnecting from Metro entirely (e.g. testing on
a physical device you're about to unplug), build the release variant instead — this bundles the
JS into the app at build time:

```sh
npx react-native run-android --mode release --device <device-id>
```

`adb devices` lists connected device IDs. Release builds are signed with the debug keystore by
default (see `android/app/build.gradle`) — fine for sideloading on your own device, not for a
real release; generate a proper keystore before shipping anywhere.

## Architecture

`App.tsx` is the actual root — it does its own auth gating and its own top-level tab switching,
rather than going through a single top-level navigator:

- `AuthProvider` wraps everything; while it's resolving the stored token, `SplashScreen` shows.
- Logged out → `LoginScreen` / `RegisterScreen` (toggled by local state, not a navigator).
- Logged in → `AssignmentsProvider` + `SubmissionsProvider`, then a custom two-tab shell (Home /
  Tasks) built directly in `App.tsx` with `BottomNavBar` — both tabs stay mounted and are
  toggled with `display: none` rather than unmounting, so switching tabs doesn't reset scroll
  position or in-flight state.
- The **Tasks** tab is the only place React Navigation is used: `TasksStackNavigator` (its own
  `NavigationContainer`) drives `AssignmentsListScreen` → `AssignmentDetailScreen` →
  `SubmissionDetailScreen`.

`src/navigation/RootNavigator.tsx` and `MainNavigator.tsx` are **not used** — leftover from an
earlier version of the navigation structure before the above was built directly into `App.tsx`.

### Directory layout

```
src/
  api/          axios client + one file per resource (assignments, submissions, auth, dashboard)
  context/      Auth/Assignments/Submissions providers — see State management above
  navigation/   TasksStackNavigator (the only real navigator) + param types
  screens/      one file per screen
  components/   shared UI (currently just BottomNavBar)
  utils/        scan.ts (camera capture), pdf.ts (fetch-as-base64 workaround, see below)
```

### Known quirks worth knowing before you touch these files

- **PDF loading**: `react-native-pdf`'s remote-URL loader stalls on anything but the first PDF
  fetched per app session (a bug in the shared `react-native-blob-util` streaming path), so
  `utils/pdf.ts` fetches PDFs as base64 and hands `react-native-pdf` a data URI instead. It also
  has no native corner-radius support on Android and paints its own gray letterbox behind the
  page — don't rely on a parent's `overflow: hidden` + `borderRadius` to clip it; pad it away
  from rounded corners instead (see `AssignmentDetailScreen`'s question-paper container).
- **Presigned URLs expire** (15 min, server-side). `AssignmentDetailScreen` and
  `SubmissionDetailScreen` both retry once by force-refreshing their data (which mints a fresh
  URL) before showing an error, since a screen left open past that window would otherwise show
  a silent blank PDF.
- **No logout UI yet** — `AuthContext.logout()` exists but no screen calls it.
- **Auth doesn't actually model students yet.** The backend's `users.role` is `teacher` or
  `admin` only — there's no `student` role, and nothing ties a logged-in account to "this is
  the student whose papers these are." The scan flow (`AssignmentDetailScreen`) reflects this:
  it asks for a student name by free-text entry every time, rather than using the logged-in
  user's own identity, and one assignment can hold submissions for several different names. If
  you're building this out for real students, that's the gap to close — see the note in the
  root README.
