export type AuthStackParamList = {
  Login: undefined;
  Register: undefined;
};

export type RootStackParamList = {
  Main: undefined;
};

export type TasksStackParamList = {
  AssignmentsList: undefined;
  AssignmentDetail: { assignmentId: string };
};
