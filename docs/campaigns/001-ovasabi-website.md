# Ovasabi Website Campaign

## Overview

This campaign launches the Ovasabi website and integrates with the Ovasabi app to drive user engagement and growth. The campaign features a waitlist, unique username reservation, a referral system, a referral leaderboard, and internationalized campaign communications.

---

## Features

### 1. Waitlist & Unique Username Reservation

- Users can sign up for the waitlist by providing their email and reserving a unique username.
- The username is globally unique and will be used as their referral code.
- Upon signup, users receive a confirmation and their reserved username.

### 2. Referral System

- Each user receives a unique referral link based on their username (e.g., `ovasabi.com/r/username`).
- Users can share their referral link to invite others to join the waitlist.
- When a new user signs up via a referral link, the referrer's count increases.

### 3. Referral Leaderboard

- The app maintains a leaderboard ranking users by the number of successful referrals.
- The leaderboard is updated in real-time and can be viewed on the campaign dashboard.
- Top referrers may receive rewards or recognition.

### 4. Campaign Information & Internationalization

- Campaign-related UI strings (e.g., banners, emails, notifications) are managed as campaign assets.
- These strings can be sent to the i18n service for translation into supported locales.
- The app dynamically serves localized campaign content based on user preference or browser settings.

---

## Data Flow & Integration

1. **User signs up:**
   - Enters email and desired username.
   - System checks username uniqueness and reserves it.
   - User is added to the waitlist and receives a referral link.

2. **Referral event:**
   - New user signs up via referral link.
   - Referrer's referral count is incremented.
   - Leaderboard is updated.

3. **Leaderboard:**
   - Aggregates referral counts for all users.
   - Ranks users and exposes the top N via API/UI.

4. **Internationalization:**
   - Campaign UI strings are sent to the i18n service for translation.
   - Translated strings are stored and served per user locale.

---

## API & Service Interactions

- **Campaign Service:** Manages campaign state, waitlist, and leaderboard.
- **User Service:** Handles user registration and username reservation.
- **Referral Service:** Tracks referrals and updates counts.
- **i18n Service:** Translates campaign strings for multi-locale support.

---

## Example User Journey

1. Alice visits ovasabi.com, signs up, and reserves `alice123`.
2. Alice shares her referral link: `ovasabi.com/r/alice123`.
3. Bob signs up using Alice's link, reserves `bobster`.
4. Alice's referral count increases; she moves up the leaderboard.
5. All campaign emails and UI are shown in the user's preferred language.

---

## User Journey (Detailed)

1. **Landing & Waitlist Signup:**
   - User visits ovasabi.com and is greeted with a campaign landing page.
   - User enters their email and desired username.
   - System checks if the username is unique and reserves it if available.
   - User is added to the waitlist and receives a confirmation email with their reserved username and unique referral link.

2. **Referral Sharing:**
   - User shares their referral link (e.g., `ovasabi.com/r/username`) via social media, email, or direct message.
   - Each new signup via the referral link increments the referrer's count.

3. **Leaderboard Engagement:**
   - Users can view a real-time leaderboard showing the top referrers.
   - The leaderboard updates as new referrals are made, encouraging competition and engagement.

4. **Localized Experience:**
   - All campaign UI strings (banners, emails, notifications) are managed as campaign assets.
   - When a user interacts with the campaign, the system detects their preferred locale (from browser or profile).
   - The i18n service is called to fetch or generate translations for all campaign strings.
   - Users see the campaign in their preferred language, increasing accessibility and reach.

5. **Live Broadcasts:**
   - The campaign can broadcast live site information (e.g., "X users joined today!", "Leaderboard just updated!", or "New milestone reached!") to all active users.
   - This is achieved via the campaign service integrating with the broadcast service, pushing real-time updates to the frontend (e.g., via websockets or server-sent events).

---

## Translations Implementation

- **Campaign assets** (UI strings, banners, emails) are registered with the i18n service.
- On campaign creation or update, all relevant strings are sent in batch to the i18n service for translation into supported locales.
- Translations are stored in the i18n database and cached in Redis for fast access.
- When a user session is initialized, the frontend requests the appropriate locale strings from the backend, which fetches from the i18n service/cache.
- If a translation is missing, the i18n service can auto-generate it using LibreTranslate and store it for future use.
- All user-facing campaign content is thus dynamically localized.

---

## Nexus Pattern for Campaign Orchestration

A Nexus pattern can be used to orchestrate the campaign workflow and ensure all services (user, referral, i18n, broadcast) interact seamlessly:

- **Pattern Definition:**
  - Define a Nexus pattern for "Campaign Waitlist & Referral Orchestration".
  - The pattern coordinates user registration, username reservation, referral tracking, leaderboard updates, and i18n translation requests.
- **Pattern Steps:**
  1. User submits signup form.
  2. Pattern checks username uniqueness via User Service.
  3. If unique, reserves username and creates user.
  4. Adds user to campaign waitlist.
  5. If referral code present, updates referral count and leaderboard.
  6. Triggers i18n translation fetch for campaign assets.
  7. Broadcasts live update to all users (e.g., new signup, leaderboard change).
- **Pattern Benefits:**
  - Centralizes campaign logic and ensures consistency across services.
  - Enables easy extension for future campaign features.

---

## Real-Time Campaign Broadcasts

- The campaign service integrates with the broadcast service to push live updates to users.
- Example broadcasts:
  - "1000th user just joined the waitlist!"
  - "Leaderboard updated: Alice123 is now #1!"
  - "New campaign milestone: Referral rewards unlocked!"
- The frontend subscribes to these broadcasts (via websockets, SSE, or polling) and updates the UI in real time.
- This creates a dynamic, engaging campaign experience and encourages viral growth.

---

## Future Enhancements

- Automated rewards for top referrers.
- Social sharing integrations.
- Advanced analytics on referral effectiveness.
- More granular localization (e.g., region-specific content).

---

## File Location

This documentation is located at `docs/campaigns/ovasabi-website.md`.

---

## API Endpoints & Features

### 1. Signup Endpoint

- `POST /api/campaigns/ovasabi-website/signup`
- Handles user waitlist signup, unique username reservation, referral tracking, i18n translation, and triggers a broadcast event.

### 2. Send Broadcast Endpoint

- `POST /api/campaigns/ovasabi-website/broadcast`
- Allows campaign admins to send live messages to all users (e.g., milestone announcements, leaderboard updates).

### 3. Referral Leaderboard Endpoint

- `GET /api/campaigns/ovasabi-website/leaderboard`
- Returns the current referral leaderboard for the campaign, ranking users by referral count.

### 4. Mouse Position Tracking (Frontend)

- The frontend tracks user mouse positions to create interactive graphic effects (e.g., animated backgrounds, engagement heatmaps).
- Mouse position data can be sent to the backend for analytics or real-time effects, but is primarily used client-side for UI/UX.

---

## Orchestration Logic

- The OvasabiWebsiteOrchestrator coordinates all campaign actions:
  - Signup: Registers user, reserves username, adds to waitlist, handles referrals, ensures translations, and broadcasts events.
  - SendBroadcast: Allows sending live campaign messages.
  - GetReferralLeaderboard: Fetches and returns the current leaderboard.
- All endpoints are exposed via the campaign API and are integrated with the relevant services (User, Campaign, Referral, i18n, Broadcast). 