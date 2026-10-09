"""Compare the room model against the CO2-only switch over the same weekdays.

Reads the days saved by eval/save_day.sh in one run's folder, as
day_model_<date>_*.json and day_switch_<date>_*.json, and prints, for each day
and as a mean over the days, the time-weighted mean damper level over the whole
day and the meeting hour, and the peak CO2. Also checks that both methods saw
the same head counts on each day, which the comparison rests on
(project_notes.md §11).

Usage: python3 eval/compare_week.py test/results/comparison_week_2026-11-23 \
    2026-11-23 [--days 5]
"""

import argparse
import json
from datetime import date, datetime, timedelta
from pathlib import Path

from day_summary import ROOM_TZ, mean_level

METHODS = ("model", "switch")


def _load(folder: Path, method: str, day: date, kind: str) -> list[dict]:
    """Rows of one saved day, as stored."""
    path = folder / f"day_{method}_{day}_{kind}.json"
    with path.open() as f:
        return json.load(f)


def _day_figures(folder: Path, method: str, day: date) -> tuple[float, float, float]:
    """Mean damper level over the day and 13:00-14:00, and peak CO2 in ppm."""
    day_start = datetime.combine(day, datetime.min.time(), tzinfo=ROOM_TZ)
    commands = [
        (datetime.fromisoformat(c["ts"]), c["level"])
        for c in _load(folder, method, day, "commands")
    ]
    whole_day = mean_level(commands, day_start, day_start + timedelta(days=1))
    meeting_start = day_start + timedelta(hours=13)
    meeting = mean_level(commands, meeting_start, meeting_start + timedelta(hours=1))
    peak = max(r["ppm"] for r in _load(folder, method, day, "co2"))
    return whole_day, meeting, peak


def _counts(folder: Path, method: str, day: date) -> list[tuple[datetime, int]]:
    """The stored head counts of one day, with their times, oldest first."""
    return [
        (datetime.fromisoformat(r["ts"]), r["count"])
        for r in _load(folder, method, day, "occupancy")
    ]


def _fits_between(
    count: tuple[datetime, int], other: list[tuple[datetime, int]]
) -> bool:
    """Whether a count lies between the other run's counts just before and after it.

    Holds for any count of the same schedule sampled at another second; a count
    with no neighbor on one side is compared with the nearest one alone.
    """
    ts, value = count
    before = [c for t, c in other if t <= ts][-1:]
    after = [c for t, c in other if t >= ts][:1]
    neighbors = before + after
    return min(neighbors) <= value <= max(neighbors)


def _same_head_counts(folder: Path, day: date) -> bool:
    """Whether both methods' head counts fit one schedule, sampled at other seconds.

    The Occupancy sensor counts once a room minute, at a second set by when the
    session started, so the same arrival can be counted up to a minute apart.
    """
    model, switch = (_counts(folder, method, day) for method in METHODS)
    model_fits = all(_fits_between(count, switch) for count in model)
    switch_fits = all(_fits_between(count, model) for count in switch)
    return model_fits and switch_fits


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("folder", type=Path, help="the run's folder of saved days")
    parser.add_argument("first", type=date.fromisoformat, help="first day, YYYY-MM-DD")
    parser.add_argument("--days", type=int, default=5)
    args = parser.parse_args()
    days = [args.first + timedelta(days=i) for i in range(args.days)]

    header = "day         method  damper day  damper meeting  peak CO2  same people"
    print(header)
    totals = {method: [0.0, 0.0, 0.0] for method in METHODS}
    for day in days:
        same = "yes" if _same_head_counts(args.folder, day) else "NO"
        for method in METHODS:
            figures = _day_figures(args.folder, method, day)
            for i, value in enumerate(figures):
                totals[method][i] += value
            whole_day, meeting, peak = figures
            print(
                f"{day}  {method:<6}  {whole_day:10.3f}  {meeting:14.3f}"
                f"  {peak:5.0f} ppm  {same}"
            )
    print(f"mean over {len(days)} days:")
    for method in METHODS:
        whole_day, meeting, peak = (total / len(days) for total in totals[method])
        print(f"  {method:<6}  {whole_day:10.3f}  {meeting:14.3f}  {peak:5.0f} ppm")


if __name__ == "__main__":
    main()
