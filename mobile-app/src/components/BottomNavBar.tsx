import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Svg, { Circle, Path, Polyline } from 'react-native-svg';

type Tab = 'tasks' | 'home';

type Props = {
  activeTab: Tab;
  onTabPress: (tab: Tab) => void;
  onCameraPress: () => void;
};

const ACTIVE_COLOR = '#2f6690';
const INACTIVE_COLOR = '#9ca3af';

function HomeIcon({ color }: { color: string }) {
  return (
    <Svg width={26} height={26} viewBox="0 0 24 24" fill="none">
      <Path
        d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <Polyline
        points="9 22 9 12 15 12 15 22"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

function TasksIcon({ color }: { color: string }) {
  return (
    <Svg width={24} height={24} viewBox="0 0 24 24" fill="none">
      <Path
        d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

export function CameraIcon({ color }: { color: string }) {
  return (
    <Svg width={26} height={26} viewBox="0 0 24 24" fill="none">
      <Path
        d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <Circle cx={12} cy={13} r={4} stroke={color} strokeWidth={2} />
    </Svg>
  );
}

export function BottomNavBar({ activeTab, onTabPress, onCameraPress }: Props) {
  const insets = useSafeAreaInsets();

  return (
    <View style={[styles.bar, { paddingBottom: Math.max(insets.bottom, 20) }]}>
      <Pressable onPress={() => onTabPress('home')} style={styles.tab}>
        <HomeIcon
          color={activeTab === 'home' ? ACTIVE_COLOR : INACTIVE_COLOR}
        />
        <Text
          style={[
            styles.label,
            { color: activeTab === 'home' ? ACTIVE_COLOR : INACTIVE_COLOR },
          ]}
        >
          Home
        </Text>
      </Pressable>

      <Pressable onPress={onCameraPress} style={styles.cameraButton}>
        <CameraIcon color="#ffffff" />
      </Pressable>

      <Pressable onPress={() => onTabPress('tasks')} style={styles.tab}>
        <TasksIcon
          color={activeTab === 'tasks' ? ACTIVE_COLOR : INACTIVE_COLOR}
        />
        <Text
          style={[
            styles.label,
            { color: activeTab === 'tasks' ? ACTIVE_COLOR : INACTIVE_COLOR },
          ]}
        >
          Tasks
        </Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-around',
    backgroundColor: '#ffffff',
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: '#e5e7eb',
    paddingTop: 14,
  },
  tab: {
    alignItems: 'center',
    justifyContent: 'center',
    width: 60,
  },
  label: {
    marginTop: 4,
    fontSize: 12,
  },
  cameraButton: {
    width: 68,
    height: 68,
    borderRadius: 34,
    backgroundColor: ACTIVE_COLOR,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: -40,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
    shadowRadius: 6,
    elevation: 6,
  },
});
