// Post dates are calendar days. Read them as UTC, so the day never shifts with the time zone.
const toDate = (date: string) => new Date(`${date}T00:00:00Z`);

const formatDate = (date: string) =>
	toDate(date).toLocaleDateString('en-GB', {
		day: 'numeric',
		month: 'long',
		year: 'numeric',
		timeZone: 'UTC'
	});

export { formatDate, toDate };
