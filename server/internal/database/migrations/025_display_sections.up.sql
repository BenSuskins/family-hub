ALTER TABLE paired_displays
ADD COLUMN enabled_sections TEXT NOT NULL DEFAULT '["calendar","meals","chores","weather","departures","trains","bins","shopping"]';
