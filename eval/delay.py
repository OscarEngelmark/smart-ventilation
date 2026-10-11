"""Measure the delay from a CO2 reading to the command it led to (NFR-4).

Reads the days saved by eval/save_day.sh in one run's folder and, for each
stored command, takes the room time from the latest CO2 reading at or before
it. Prints the number of commands and the median and longest delay, in room
seconds, for each method and for both together (project_notes.md §11).

Usage: python3 eval/delay.py test/results/comparison_week_2026-11-23 \
    2026-11-23 [--days 5]
"""

import argparse
import bisect
import statistics
from datetime import date, datetime, timedelta
from pathlib import Path

from compare_week import METHODS, load_day


def _times(rows: list[dict]) -> list[datetime]:
    """The stored rows' times, oldest first."""
    return [datetime.fromisoformat(r["ts"]) for r in rows]


def day_delays(folder: Path, method: str, day: date) -> list[float]:
    """Room seconds from each command of one day back to the reading before it.

    A command with no reading before it on that day, such as the one in force
    when the day began, is left out.
    """
    readings = _times(load_day(folder, method, day, "co2"))
    delays = []
    for command in _times(load_day(folder, method, day, "commands")):
        i = bisect.bisect_right(readings, command)  # readings[:i] are at or before it
        if i == 0:
            continue
        delays.append((command - readings[i - 1]).total_seconds())
    return delays


def _summary(label: str, delays: list[float]) -> str:
    """One line: the number of commands and the median and longest delay."""
    median = statistics.median(delays)
    return f"{label:<6}  {len(delays):8d}  {median:8.0f} s  {max(delays):7.0f} s"


def main() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("folder", type=Path, help="the run's folder of saved days")
    parser.add_argument("first", type=date.fromisoformat, help="first day, YYYY-MM-DD")
    parser.add_argument("--days", type=int, default=5)
    args = parser.parse_args()
    days = [args.first + timedelta(days=i) for i in range(args.days)]

    print("method  commands    median    longest")
    every = []
    for method in METHODS:
        delays = [d for day in days for d in day_delays(args.folder, method, day)]
        every += delays
        print(_summary(method, delays))
    print(_summary("both", every))


if __name__ == "__main__":
    main()
