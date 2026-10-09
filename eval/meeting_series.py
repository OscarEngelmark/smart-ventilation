"""Write the CO2 and damper level around one day's meeting, per minute, for a plot.

Reads one day of both methods from a run's folder (see eval/compare_week.py) and
writes a whitespace-separated table, one row per room minute from 12:00 to 15:00:
the hour of day, then for the room model and for the CO2-only switch the mean CO2
reading of that minute in ppm and the damper level in force at its start. The
report's meeting plot reads this table.

Usage: python3 eval/meeting_series.py test/results/comparison_week_2026-11-23 \
    2026-11-27 latex/report/data/meeting_2026-11-27.dat
"""

import argparse
from datetime import date, datetime, timedelta
from pathlib import Path

from compare_week import METHODS, load_day
from day_summary import ROOM_TZ

FIRST_HOUR = 12
LAST_HOUR = 15


def _minute_co2(folder: Path, method: str, day: date) -> dict[datetime, float]:
    """Mean CO2 reading in ppm of each room minute, keyed by the minute's start."""
    readings: dict[datetime, list[float]] = {}
    for r in load_day(folder, method, day, "co2"):
        ts = datetime.fromisoformat(r["ts"]).astimezone(ROOM_TZ)
        minute = ts.replace(second=0, microsecond=0)
        readings.setdefault(minute, []).append(r["ppm"])
    return {minute: sum(ppm) / len(ppm) for minute, ppm in readings.items()}


def _level_at(commands: list[tuple[datetime, float]], moment: datetime) -> float:
    """Damper level in force at a moment, from (time, level) pairs, oldest first."""
    in_force = [level for ts, level in commands if ts <= moment]
    return in_force[-1]


def _series(folder: Path, method: str, day: date, minutes: list[datetime]) -> list[str]:
    """One method's CO2 and damper level for each minute, as formatted columns."""
    co2 = _minute_co2(folder, method, day)
    commands = [
        (datetime.fromisoformat(c["ts"]), c["level"])
        for c in load_day(folder, method, day, "commands")
    ]
    return [f"{co2[m]:7.1f} {_level_at(commands, m):5.2f}" for m in minutes]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("folder", type=Path, help="the run's folder of saved days")
    parser.add_argument("day", type=date.fromisoformat, help="room date, YYYY-MM-DD")
    parser.add_argument("out", type=Path, help="table file to write")
    args = parser.parse_args()

    day_start = datetime.combine(args.day, datetime.min.time(), tzinfo=ROOM_TZ)
    first = day_start + timedelta(hours=FIRST_HOUR)
    count = (LAST_HOUR - FIRST_HOUR) * 60
    minutes = [first + timedelta(minutes=i) for i in range(count)]
    columns = {m: _series(args.folder, m, args.day, minutes) for m in METHODS}

    lines = ["hour co2_model damper_model co2_switch damper_switch"]
    for i, minute in enumerate(minutes):
        hour = minute.hour + minute.minute / 60
        lines.append(f"{hour:.4f} {columns['model'][i]} {columns['switch'][i]}")
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text("\n".join(lines) + "\n")


if __name__ == "__main__":
    main()
