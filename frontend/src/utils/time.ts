export function getDaysDifference(dateStr: string | undefined): number {
  if (!dateStr) return 0;
  const completedDate = new Date(dateStr);
  const now = new Date();
  const diffTime = now.getTime() - completedDate.getTime();
  return Math.floor(diffTime / (1000 * 60 * 60 * 24));
}

export function getHoursDifference(dateStr: string | undefined): number {
  if (!dateStr) return 0;
  const completedDate = new Date(dateStr);
  const now = new Date();
  const diffTime = now.getTime() - completedDate.getTime();
  return Math.floor(diffTime / (1000 * 60 * 60));
}
