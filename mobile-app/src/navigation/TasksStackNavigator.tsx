import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { AssignmentsListScreen } from '../screens/AssignmentsListScreen';
import { AssignmentDetailScreen } from '../screens/AssignmentDetailScreen';
import { TasksStackParamList } from './types';

const Stack = createNativeStackNavigator<TasksStackParamList>();

export function TasksStackNavigator() {
  return (
    <Stack.Navigator>
      <Stack.Screen
        name="AssignmentsList"
        component={AssignmentsListScreen}
        options={{ title: 'Assignments' }}
      />
      <Stack.Screen
        name="AssignmentDetail"
        component={AssignmentDetailScreen}
        options={{ title: 'Assignment' }}
      />
    </Stack.Navigator>
  );
}
