export function formatSubscriptionDuration(plan, t) {
  const unit = plan?.duration_unit || 'month';
  const value = plan?.duration_value || 1;
  const unitLabels = {
    year: t('年'),
    month: t('个月'),
    day: t('天'),
    hour: t('小时'),
    custom: t('自定义'),
  };
  if (unit === 'custom') {
    const seconds = plan?.custom_seconds || 0;
    if (seconds >= 86400) return `${Math.floor(seconds / 86400)} ${t('天')}`;
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)} ${t('小时')}`;
    return `${seconds} ${t('秒')}`;
  }
  return `${value} ${unitLabels[unit] || unit}`;
}

export function formatSubscriptionResetPeriod(plan, t) {
  const period = plan?.quota_reset_period || 'never';
  if (period === 'never') return t('不重置');
  if (period === 'daily') return t('每天');
  if (period === 'weekly') return t('每周');
  if (period === 'monthly') return t('每月');
  if (period === 'custom') {
    const seconds = Number(plan?.quota_reset_custom_seconds || 0);
    if (seconds >= 86400) return `${Math.floor(seconds / 86400)} ${t('天')}`;
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)} ${t('小时')}`;
    if (seconds >= 60) return `${Math.floor(seconds / 60)} ${t('分钟')}`;
    return `${seconds} ${t('秒')}`;
  }
  return t('不重置');
}

function addSubscriptionDuration(start, plan) {
  const unit = plan?.duration_unit || 'month';
  const value = Number(plan?.duration_value || 1);
  const end = new Date(start.getTime());
  if (unit === 'custom') {
    end.setSeconds(end.getSeconds() + Number(plan?.custom_seconds || 0));
    return end;
  }
  if (unit === 'year') {
    end.setFullYear(end.getFullYear() + value);
    return end;
  }
  if (unit === 'month') {
    end.setMonth(end.getMonth() + value);
    return end;
  }
  if (unit === 'day') {
    end.setDate(end.getDate() + value);
    return end;
  }
  if (unit === 'hour') {
    end.setHours(end.getHours() + value);
    return end;
  }
  end.setMonth(end.getMonth() + value);
  return end;
}

function nextSubscriptionResetTime(base, plan) {
  const period = plan?.quota_reset_period || 'never';
  const next = new Date(base.getTime());
  if (period === 'daily') {
    next.setHours(0, 0, 0, 0);
    next.setDate(next.getDate() + 1);
    return next;
  }
  if (period === 'weekly') {
    next.setHours(0, 0, 0, 0);
    const day = next.getDay() || 7;
    next.setDate(next.getDate() + (8 - day));
    return next;
  }
  if (period === 'monthly') {
    next.setHours(0, 0, 0, 0);
    next.setDate(1);
    next.setMonth(next.getMonth() + 1);
    return next;
  }
  if (period === 'custom') {
    const seconds = Number(plan?.quota_reset_custom_seconds || 0);
    if (seconds <= 0) return null;
    next.setSeconds(next.getSeconds() + seconds);
    return next;
  }
  return null;
}

export function calculateSubscriptionTotalQuota(plan, start = new Date()) {
  const periodQuota = Number(plan?.total_amount || 0);
  if (periodQuota <= 0) return 0;
  if ((plan?.quota_reset_period || 'never') === 'never') return periodQuota;

  const end = addSubscriptionDuration(start, plan);
  if (!(end instanceof Date) || Number.isNaN(end.getTime()) || end <= start) {
    return periodQuota;
  }

  let periods = 1;
  let cursor = start;
  let next = nextSubscriptionResetTime(cursor, plan);
  while (next && next < end && periods < 10000) {
    periods += 1;
    cursor = next;
    next = nextSubscriptionResetTime(cursor, plan);
  }
  return periodQuota * periods;
}
