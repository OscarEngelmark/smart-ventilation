"""Summarize one room day from the storage-service: damper use and peak CO2.

Prints the time-weighted mean damper level over the whole day, the hour before
the meeting, and the meeting hour, and the highest stored CO2 reading. Used to
compare a day planned with the occupancy forecast against a day planned
without it (project_notes.md §7).

Usage: python3 eval/day_summary.py 2026-10-05 [--room level0/A125]
"""

import argparse
import json
import urllib.parse
import urllib.request
from datetime import date, datetime, timedelta
from zoneinfo import ZoneInfo

STORAGE_URL = "http://localhost:8081"
ROOM_TZ = ZoneInfo("Europe/Stockholm")

# Label, start hour and end hour of each window, in room time.
WINDOWS = [
    ("whole day", 0, 24),
    ("12:00-13:00, before meeting", 12, 13),
    ("13:00-14:00, meeting", 13, 14),
]


def _fetch(path: str, room: str, since: datetime) -> list[dict]:
    """Get a stored history from the storage-service, oldest first."""
    query = urllib.parse.urlencode({"room": room, "since": since.isoformat()})
    with urllib.request.urlopen(f"{STORAGE_URL}{path}?{query}") as resp:
        return json.load(resp)


def mean_level(
    commands: list[tuple[datetime, float]], start: datetime, end: datetime
) -> float:
    """Time-weighted mean damper level between start and end.

    Parameters
    ----------
    commands — (time, level) pairs, oldest first; each level holds from its
        time until the next command
    start — beginning of the window; a command must be in force at it
    end — end of the window

    Returns
    -------
    The mean level, 0 (closed) to 1 (fully open).
    """
    in_force = [level for ts, level in commands if ts <= start]
    if not in_force:
        raise ValueError(f"no damper command in force at {start}")
    level = in_force[-1]
    t = start
    weighted = 0.0
    for ts, next_level in commands:
        if ts <= start:
            continue
        if ts >= end:
            break
        weighted += level * (ts - t).total_seconds()
        t, level = ts, next_level
    weighted += level * (end - t).total_seconds()
    return weighted / (end - start).total_seconds()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("day", type=date.fromisoformat, help="room date, YYYY-MM-DD")
    parser.add_argument("--room", default="level0/A125")
    args = parser.parse_args()

    day_start = datetime.combine(args.day, datetime.min.time(), tzinfo=ROOM_TZ)
    day_end = day_start + timedelta(days=1)

    stored_commands = _fetch("/commands", args.room, day_start)
    commands = [(datetime.fromisoformat(c["ts"]), c["level"]) for c in stored_commands]
    stored_readings = _fetch("/co2", args.room, day_start)
    readings = [
        (datetime.fromisoformat(r["ts"]), r["ppm"])
        for r in stored_readings
        if datetime.fromisoformat(r["ts"]) < day_end
    ]
    if not readings:
        raise SystemExit(f"no CO2 readings stored for {args.room} on {args.day}")

    first = readings[0][0].astimezone(ROOM_TZ)
    last = readings[-1][0].astimezone(ROOM_TZ)
    print(f"{args.room}, {args.day}: {len(readings)} readings")
    print(f"  from {first:%H:%M} to {last:%H:%M}")
    print("mean damper level:")
    for label, start_hour, end_hour in WINDOWS:
        start = day_start + timedelta(hours=start_hour)
        end = day_start + timedelta(hours=end_hour)
        print(f"  {label}: {mean_level(commands, start, end):.3f}")
    peak_ts, peak_ppm = max(readings, key=lambda r: r[1])
    print(f"peak CO2: {peak_ppm:.0f} ppm at {peak_ts.astimezone(ROOM_TZ):%H:%M}")


if __name__ == "__main__":
    main()
