import { useEffect, useState } from 'react';
import { ActivityIndicator, Pressable, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Svg, { Circle, Path, Polyline } from 'react-native-svg';
import { useAuth } from '../context/AuthContext';
import {
  DailyActivity,
  DashboardSummary,
  fetchDashboardSummary,
} from '../api/dashboard';

const ACCENT_COLOR = '#2f6690';

function BellIcon({ color }: { color: string }) {
  return (
    <Svg width={20} height={20} viewBox="0 0 24 24" fill="none">
      <Path
        d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <Path
        d="M13.73 21a2 2 0 0 1-3.46 0"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

function ActivitySparkline({ data }: { data: DailyActivity[] }) {
  const width = 260;
  const height = 56;
  const max = Math.max(1, ...data.map(d => d.pages_scanned));
  const stepX = data.length > 1 ? width / (data.length - 1) : 0;
  const points = data.map((d, i) => ({
    x: i * stepX,
    y: height - (d.pages_scanned / max) * height,
  }));
  const polylinePoints = points.map(p => `${p.x},${p.y}`).join(' ');

  return (
    <Svg width={width} height={height} style={{ marginTop: 12 }}>
      <Polyline
        points={polylinePoints}
        fill="none"
        stroke={ACCENT_COLOR}
        strokeWidth={2}
      />
      {points.map((p, i) => (
        <Circle key={i} cx={p.x} cy={p.y} r={3} fill={ACCENT_COLOR} />
      ))}
    </Svg>
  );
}

function WeekStrip({ data }: { data: DailyActivity[] }) {
  const todayIndex = data.length - 1;
  return (
    <View className="mt-4 flex-row justify-between">
      {data.map((d, i) => {
        const isToday = i === todayIndex;
        const hasActivity = d.pages_scanned > 0;
        return (
          <View key={d.date} className="items-center">
            <Text className="mb-2 text-xs text-gray-400">{d.weekday}</Text>
            <View
              className="h-9 w-9 items-center justify-center rounded-full"
              style={{
                backgroundColor: isToday
                  ? ACCENT_COLOR
                  : hasActivity
                    ? '#dbe6ec'
                    : '#f3f4f6',
              }}
            >
              <Text
                style={{
                  color: isToday ? '#ffffff' : hasActivity ? ACCENT_COLOR : '#d1d5db',
                }}
              >
                {hasActivity ? '✓' : '·'}
              </Text>
            </View>
          </View>
        );
      })}
    </View>
  );
}

function ProgressBubbles() {
  return (
    <View className="w-20 flex-row flex-wrap justify-end">
      {[...Array(6)].map((_, i) => (
        <View
          key={i}
          className="m-0.5 h-6 w-6 rounded-full"
          style={{ borderWidth: 1, borderColor: 'rgba(255,255,255,0.3)' }}
        />
      ))}
    </View>
  );
}

export function HomeScreen() {
  const { user } = useAuth();
  const insets = useSafeAreaInsets();
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [status, setStatus] = useState<'loading' | 'loaded' | 'error'>('loading');

  useEffect(() => {
    fetchDashboardSummary()
      .then(data => {
        setSummary(data);
        setStatus('loaded');
      })
      .catch(() => setStatus('error'));
  }, []);

  if (status === 'loading') {
    return (
      <View className="flex-1 items-center justify-center bg-white">
        <ActivityIndicator size="large" color={ACCENT_COLOR} />
      </View>
    );
  }

  if (status === 'error' || !summary) {
    return (
      <View className="flex-1 items-center justify-center bg-white px-6">
        <Text className="text-center text-sm text-red-600">
          Failed to load dashboard.
        </Text>
      </View>
    );
  }

  const initial = user?.email ? user.email[0].toUpperCase() : '?';

  return (
    <View className="flex-1 bg-white">
      <View
        className="flex-row items-center justify-between px-6"
        style={{ paddingTop: Math.max(insets.top, 16) + 12 }}
      >

        <View className="flex-row items-center">
          <View
            className="h-12 w-12 items-center justify-center rounded-full"
            style={{ backgroundColor: ACCENT_COLOR }}
          >
            <Text className="text-lg font-semibold text-white">{initial}</Text>
          </View>
          <Text className="ml-3 text-xl font-bold text-gray-900">Dashboard</Text>
        </View>
        <View className="flex-row items-center">
          <Pressable className="h-11 w-11 items-center justify-center rounded-full bg-gray-100">
            <BellIcon color="#374151" />
          </Pressable>
          <Pressable
            onPress={() => {}}
            className="ml-3 h-11 w-11 items-center justify-center rounded-xl"
            style={{ backgroundColor: ACCENT_COLOR }}
          >
            <Text className="text-2xl font-light text-white">+</Text>
          </Pressable>
        </View>
      </View>

      <View className="px-6 pb-6 pt-6">
        <View className="rounded-3xl bg-gray-50 p-5">
          <Text className="text-xs font-semibold tracking-wide text-gray-400">
            THIS WEEK'S SCANS
          </Text>
          <View className="items-center">
            <View className="mt-3 rounded-full bg-gray-900 px-4 py-1.5">
              <Text className="text-sm font-semibold text-white">
                {summary.today_pages_scanned}{' '}
                {summary.today_pages_scanned === 1 ? 'page' : 'pages'} today
              </Text>
            </View>
            <ActivitySparkline data={summary.weekly_activity} />
          </View>
          <WeekStrip data={summary.weekly_activity} />
        </View>

        <View className="mt-4 flex-row items-center justify-between rounded-3xl bg-gray-900 p-5">
          <View>
            <Text className="text-xs font-semibold tracking-wide text-gray-400">
              GRADING PROGRESS
            </Text>
            <Text className="mt-2 text-4xl font-bold text-white">
              {Math.round(summary.composited_percentage)}%
            </Text>
            <Text className="mt-1 text-xs text-gray-400">
              Of this week's submissions{'\n'}composited
            </Text>
          </View>
          <ProgressBubbles />
        </View>

        <View className="mt-4 rounded-3xl bg-gray-50 p-5">
          <View className="flex-row items-center justify-between">
            <Text className="text-xs font-semibold tracking-wide text-gray-400">
              SUBMISSIONS
            </Text>
            <Text className="text-xs text-gray-500">
              {summary.submissions_this_week} sessions
            </Text>
          </View>
          <Text className="mt-2 text-3xl font-bold text-gray-900">
            {summary.pages_scanned_this_week}{' '}
            <Text className="text-base font-normal text-gray-400">
              {summary.pages_scanned_this_week === 1 ? 'page' : 'pages'} /this
              week
            </Text>
          </Text>
        </View>
      </View>
    </View>
  );
}
