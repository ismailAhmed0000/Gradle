import { useEffect, useState } from 'react';
import { ActivityIndicator, Pressable, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Svg, { Path } from 'react-native-svg';
import { useAuth } from '../context/AuthContext';
import { DashboardSummary, fetchDashboardSummary } from '../api/dashboard';

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

export function HomeScreen() {
  const { user } = useAuth();
  const insets = useSafeAreaInsets();
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [status, setStatus] = useState<'loading' | 'loaded' | 'error'>(
    'loading',
  );

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

  return (
    <View className="flex-1 bg-white">
      <View
        className="flex-row items-center justify-between px-6"
        style={{ paddingTop: Math.max(insets.top, 16) + 12 }}
      >
        <View>
          <Text className="text-xl font-bold text-gray-900">Welcome,</Text>
          <Text className="text-xl font-bold text-gray-900">{user?.email}</Text>
        </View>
        <Pressable className="h-11 w-11 items-center justify-center rounded-full bg-gray-100">
          <BellIcon color="#374151" />
        </Pressable>
      </View>

      <View className="px-6 pb-6 pt-6">
        <View className="rounded-3xl bg-gray-900 p-7">
          <Text className="text-xs font-semibold tracking-wide text-gray-400">
            ANALYTICS
          </Text>
          <View className="mt-6 flex-row justify-center gap-16">
            <View className="items-center">
              <Text className="text-5xl font-bold text-white">
                {summary.submitted_this_week}
              </Text>
              <Text className="mt-2 text-sm text-gray-400">Submitted</Text>
            </View>
            <View className="items-center">
              <Text className="text-5xl font-bold text-white">
                {summary.pending_this_week}
              </Text>
              <Text className="mt-2 text-sm text-gray-400">Pending</Text>
            </View>
          </View>
        </View>

        <View className="mt-4 rounded-3xl bg-gray-50 p-5">
          <View className="flex-row items-center justify-between">
            <Text className="text-xs font-semibold tracking-wide text-gray-400">
              RECENT SUBMISSIONS
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
