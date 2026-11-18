r"""Browser test for card disconnect handling (Windows; needs Playwright + Chrome).

Mount a scratch copy of a few clips as a drive letter, then start the app on it:
    subst Q: C:\path\to\fakecard
    set DASHCAM_STUDIO_HOME=%TEMP%\dm-test & go run ./cmd/dashcam-studio Q:\ --ui none --port 8770
Then:
    python tests/e2e/disconnect_chrome.py Q C:\path\to\fakecard [screenshot_dir]

Removing the drive letter mid-playback simulates pulling the card; re-adding it
simulates plugging it back in. Never point this at the real card.
"""
import asyncio
import json
import os
import subprocess
import sys
import time

from playwright.async_api import async_playwright

LETTER, FAKE = sys.argv[1], sys.argv[2]
OUT = sys.argv[3] if len(sys.argv) > 3 else "."
BASE = os.environ.get("DASHCAM_URL", "http://localhost:8770")
DM = "window.dashcamStudio"


def subst(on: bool):
    args = ["subst", f"{LETTER}:", FAKE] if on else ["subst", f"{LETTER}:", "/D"]
    subprocess.run(args, check=True)


async def main():
    async with async_playwright() as p:
        b = await p.chromium.launch(channel="chrome", headless=False, args=["--autoplay-policy=no-user-gesture-required"])
        ctx = await b.new_context(viewport={"width": 1440, "height": 860}, accept_downloads=True)
        pg = await ctx.new_page()
        errs = []
        pg.on("pageerror", lambda e: errs.append(str(e)))
        await pg.request.post(BASE + "/api/open", data=json.dumps({"path": f"{LETTER}:\\"}), headers={"Content-Type": "application/json"})
        await pg.goto(BASE + "/")
        await pg.wait_for_function(f"{DM}.store.getState().current?.detail", timeout=15000)
        await pg.wait_for_timeout(800)
        await pg.evaluate(f"{DM}.engine.seek(20); {DM}.engine.play()")
        await pg.wait_for_timeout(4000)
        before = await pg.evaluate(f"({{id: {DM}.store.getState().current.id, t: {DM}.engine.now()}})")
        print("playing", before)

        # ---- pull the card
        subst(False)
        t0 = time.time()
        await pg.wait_for_selector(".media-lost", timeout=10000)
        print(f"lost detected after {time.time() - t0:.1f}s | playing:", await pg.evaluate(f"{DM}.engine.playing"))
        await pg.wait_for_timeout(1500)
        await pg.screenshot(path=os.path.join(OUT, "40_media_lost.png"))

        # ---- plug it back in
        subst(True)
        t0 = time.time()
        await pg.wait_for_selector(".media-lost", state="detached", timeout=15000)
        await pg.wait_for_timeout(2500)
        after = await pg.evaluate(f"({{id: {DM}.store.getState().current.id, t: {DM}.engine.now(), playing: {DM}.engine.playing}})")
        print(f"reconnected after {time.time() - t0:.1f}s | resumed:", after)
        assert after["id"] == before["id"] and after["playing"] and after["t"] >= before["t"] - 1, "should resume where it stopped"
        await pg.screenshot(path=os.path.join(OUT, "41_reconnected.png"))

        # ---- user disconnect (E key opens the dialog)
        await pg.keyboard.press("e")
        await pg.wait_for_selector(".disconnect-card")
        await pg.wait_for_timeout(700)  # let the dialog animate in
        await pg.screenshot(path=os.path.join(OUT, "42_disconnect_dialog.png"))
        await pg.click(".disconnect-card footer .btn.primary")
        await pg.wait_for_selector(".welcome .notice", timeout=15000)
        await pg.wait_for_timeout(1000)
        print("notice:", (await pg.inner_text(".welcome .notice")).replace("\n", " | "))
        await pg.screenshot(path=os.path.join(OUT, "43_after_disconnect.png"))
        # nothing on the card is held open any more: the drive letter can be removed and re-added
        subst(False)
        subst(True)
        print("errors:", errs[:5])
        await b.close()


asyncio.run(main())
