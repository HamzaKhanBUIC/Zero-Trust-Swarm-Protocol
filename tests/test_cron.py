from unittest.mock import patch
from src.modules.scheduler.cron import schedule_sync_job

@patch("src.modules.scheduler.cron.BackgroundScheduler")
def test_schedule_sync_job_adds_job_and_starts(mock_scheduler_class):
    mock_scheduler = mock_scheduler_class.return_value
    
    # We will pass a dummy function to the scheduler
    def dummy_job():
        pass
        
    schedule_sync_job(dummy_job, hours=12)
    
    # Verify the job was added with the correct interval
    mock_scheduler.add_job.assert_called_once()
    kwargs = mock_scheduler.add_job.call_args[1]
    assert kwargs["trigger"] == "interval"
    assert kwargs["hours"] == 12
    
    # Verify the scheduler was started
    mock_scheduler.start.assert_called_once()
