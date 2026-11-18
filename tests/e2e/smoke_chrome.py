r"""Browser smoke test for the React UI (needs: pip install playwright, and Google Chrome).

Start the app first, headless (use a scratch config dir so test favorites don't touch yours):
    set DASHCAM_STUDIO_HOME=%TEMP%\dm-test & go run ./cmd/dashcam-studio E:\ --ui none
Then:
    python tests/e2e/smoke_chrome.py [screenshot_dir]

Set DASHCAM_URL if the app picked another port (default http://localhost:8765).
"""
import asyncio
import json
import os
import sys
from urllib.parse import quote

from playwright.async_api import async_playwright

OUT = sys.argv[1] if len(sys.argv) > 1 else "."
BASE = os.environ.get("DASHCAM_URL", "http://localhost:8765")
DM = "window.dashcamStudio"


async def main():
    async with async_playwright() as p:
        b = await p.chromium.launch(channel="chrome", headless=False, args=["--autoplay-policy=no-user-gesture-required"])
        ctx = await b.new_context(viewport={"width": 1600, "height": 950}, accept_downloads=True, locale="he-IL")
        pg = await ctx.new_page()
        errs = []
        pg.on("pageerror", lambda e: errs.append(str(e)))
        pg.on("console", lambda m: m.type == "error" and errs.append(m.text))
        lib = await (await pg.request.get(BASE + "/api/library")).json()
        gclip = next(c for c in lib["clips"] if c.get("gps") and c["gps"]["km"] > 0.3)
        print("clip", gclip["id"], gclip["gps"])
        await pg.goto(BASE + "/#clip=" + quote(gclip["id"]))
        await pg.wait_for_selector(".clip.active", timeout=15000)
        print("dir:", await pg.evaluate("document.documentElement.dir"), "| title:", await pg.title())

        days = await pg.eval_on_selector_all(".lib-list section[data-day]", "els => els.map(e => e.dataset.day)")
        print("days newest->oldest:", days[:4], "sorted desc:", days == sorted(days, reverse=True))
        rows = await pg.eval_on_selector_all(".trip.open .clip", "els => els.map(e => e.querySelector('.clip-time span').textContent)")
        print("open-trip rows (top->bottom):", rows[:4], "desc:", rows == sorted(rows, reverse=True))

        await pg.wait_for_timeout(1200)
        await pg.keyboard.press("Space")
        await pg.wait_for_timeout(5000)
        drift = await pg.evaluate(f"(() => {{ const e = {DM}.engine; const m = e.now(); return ['R','C'].map(c => +(e.players[c].time - m).toFixed(3)); }})()")
        print("playing:", await pg.evaluate(f"{DM}.engine.playing"), "| sync drift R,C:", drift)
        print("speedo:", await pg.inner_text(".speedo b"), "| coords:", await pg.inner_text(".coords span"))
        await pg.screenshot(path=os.path.join(OUT, "21_player_rtl.png"))

        # star with the keyboard (*), check the favorites tab
        await pg.keyboard.press("Shift+8")
        await pg.wait_for_timeout(600)
        favs = await (await pg.request.get(BASE + "/api/favorites")).json()
        print("favorites after *:", favs["ids"])
        await pg.click(".fav-tab")
        await pg.wait_for_timeout(800)
        print("favorites rows:", await pg.locator(".fav-list .clip").count())
        await pg.screenshot(path=os.path.join(OUT, "22_favorites.png"))

        # back to the drive list; select two clips with checkboxes (2nd with shift for a range)
        await pg.click(".cat-tabs button >> nth=0")
        await pg.wait_for_timeout(500)
        boxes = pg.locator(".trip.open .clip .check")
        await boxes.nth(0).click()
        await boxes.nth(2).click(modifiers=["Shift"])
        sel = await pg.evaluate(f"{DM}.store.getState().selection.size")
        print("selected via shift-range:", sel, "| bar:", await pg.inner_text(".sel-bar .sel-count"))
        await pg.screenshot(path=os.path.join(OUT, "23_selection.png"))
        # star the selection, then export it
        await pg.click(".sel-bar .icon-btn >> nth=0")
        await pg.wait_for_timeout(500)
        await pg.click(".sel-bar .btn.primary")
        await pg.wait_for_selector(".export-card")
        await pg.click(".cam-pick.cam-C")  # drop the cabin camera
        await pg.screenshot(path=os.path.join(OUT, "24_export.png"))
        total = await pg.inner_text(".export-total b")
        async with pg.expect_download() as dl:
            await pg.click(".export-card footer .btn.primary")
        d = await dl.value
        print("export:", d.suggested_filename, "| shown total:", total)
        await d.cancel()
        await pg.keyboard.press("Escape")

        # favorites view now has 3+ clips -> export all
        await pg.click(".fav-tab")
        await pg.wait_for_timeout(500)
        print("favorites now:", await pg.locator(".fav-list .clip").count(), "|", await pg.inner_text(".fav-export"))
        await pg.screenshot(path=os.path.join(OUT, "25_favorites_many.png"))

        # download menu + snapshot still work
        await pg.click(".dl-btn")
        await pg.screenshot(path=os.path.join(OUT, "26_dl_menu.png"))
        async with pg.expect_download() as dl:
            await pg.click(".menu .hero")
        d = await dl.value
        print("clip zip:", d.suggested_filename)
        await d.cancel()
        async with pg.expect_download() as dl:
            await pg.keyboard.press("s")
        d = await dl.value
        await d.save_as(os.path.join(OUT, "27_snapshot.png"))
        print("snapshot:", d.suggested_filename)

        for w, h in ((1920, 1080), (1366, 768), (1100, 800)):
            await pg.set_viewport_size({"width": w, "height": h})
            await pg.wait_for_timeout(700)
            await pg.screenshot(path=os.path.join(OUT, f"28_{w}.png"))
        await pg.evaluate(f"{DM}.engine.pause()")
        await pg.set_viewport_size({"width": 1366, "height": 768})
        await pg.click(".root-chip")
        await pg.wait_for_timeout(1200)
        await pg.screenshot(path=os.path.join(OUT, "29_welcome.png"))

        # clean up test favorites
        ids = (await (await pg.request.get(BASE + "/api/favorites")).json())["ids"]
        await pg.request.post(BASE + "/api/favorites", data=json.dumps({"ids": ids, "on": False}), headers={"Content-Type": "application/json"})
        print("errors:", errs[:10])
        await b.close()


asyncio.run(main())
