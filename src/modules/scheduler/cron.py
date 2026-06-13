from apscheduler.schedulers.background import BackgroundScheduler
from typing import Callable

def schedule_sync_job(job_func: Callable, hours: int = 12):
    """
    Schedules a given function to run at the specified interval in hours using APScheduler.
    Starts the background scheduler immediately.
    """
    scheduler = BackgroundScheduler()
    scheduler.add_job(job_func, trigger='interval', hours=hours)
    scheduler.start()
    return scheduler
