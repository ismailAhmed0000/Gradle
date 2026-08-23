import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
} from 'react';
import {
  CompositedResult,
  fetchCompositedDocument,
  listSubmissions,
  SubmissionSummary,
} from '../api/submissions';

type FetchStatus = 'idle' | 'loading' | 'loaded' | 'error';

const POLL_INTERVAL_MS = 3000;

type State = {
  submissionsByAssignmentId: Record<string, SubmissionSummary[]>;
  submissionsStatusByAssignmentId: Record<string, FetchStatus>;
  submissionsErrorByAssignmentId: Record<string, string | null>;
  compositedBySubmissionId: Record<string, CompositedResult>;
};

type Action =
  | { type: 'SUBMISSIONS_LOADING'; assignmentId: string }
  | {
      type: 'SUBMISSIONS_LOADED';
      assignmentId: string;
      payload: SubmissionSummary[];
    }
  | { type: 'SUBMISSIONS_ERROR'; assignmentId: string; payload: string }
  | {
      type: 'COMPOSITED_LOADED';
      submissionId: string;
      payload: CompositedResult;
    };

const initialState: State = {
  submissionsByAssignmentId: {},
  submissionsStatusByAssignmentId: {},
  submissionsErrorByAssignmentId: {},
  compositedBySubmissionId: {},
};

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'SUBMISSIONS_LOADING':
      return {
        ...state,
        submissionsStatusByAssignmentId: {
          ...state.submissionsStatusByAssignmentId,
          [action.assignmentId]: 'loading',
        },
        submissionsErrorByAssignmentId: {
          ...state.submissionsErrorByAssignmentId,
          [action.assignmentId]: null,
        },
      };
    case 'SUBMISSIONS_LOADED':
      return {
        ...state,
        submissionsByAssignmentId: {
          ...state.submissionsByAssignmentId,
          [action.assignmentId]: action.payload,
        },
        submissionsStatusByAssignmentId: {
          ...state.submissionsStatusByAssignmentId,
          [action.assignmentId]: 'loaded',
        },
      };
    case 'SUBMISSIONS_ERROR':
      return {
        ...state,
        submissionsStatusByAssignmentId: {
          ...state.submissionsStatusByAssignmentId,
          [action.assignmentId]: 'error',
        },
        submissionsErrorByAssignmentId: {
          ...state.submissionsErrorByAssignmentId,
          [action.assignmentId]: action.payload,
        },
      };
    case 'COMPOSITED_LOADED':
      return {
        ...state,
        compositedBySubmissionId: {
          ...state.compositedBySubmissionId,
          [action.submissionId]: action.payload,
        },
      };
    default:
      return state;
  }
}

type SubmissionsContextValue = {
  submissionsByAssignmentId: Record<string, SubmissionSummary[]>;
  submissionsStatusByAssignmentId: Record<string, FetchStatus>;
  submissionsErrorByAssignmentId: Record<string, string | null>;
  loadSubmissions: (assignmentId: string, force?: boolean) => Promise<void>;
  compositedBySubmissionId: Record<string, CompositedResult>;
  startPollingComposited: (submissionId: string) => void;
  stopPollingComposited: (submissionId: string) => void;
};

const SubmissionsContext = createContext<SubmissionsContextValue | undefined>(
  undefined,
);

export function SubmissionsProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [state, dispatch] = useReducer(reducer, initialState);

  const stateRef = useRef(state);
  stateRef.current = state;

  const pollTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>(
    {},
  );

  useEffect(() => {
    const timers = pollTimers.current;
    return () => {
      Object.values(timers).forEach(clearTimeout);
    };
  }, []);

  const loadSubmissions = useCallback(
    async (assignmentId: string, force = false) => {
      const status = stateRef.current.submissionsStatusByAssignmentId[assignmentId];
      if (!force && (status === 'loading' || status === 'loaded')) {
        return;
      }
      dispatch({ type: 'SUBMISSIONS_LOADING', assignmentId });
      try {
        const submissions = await listSubmissions(assignmentId);
        dispatch({ type: 'SUBMISSIONS_LOADED', assignmentId, payload: submissions });
      } catch {
        dispatch({
          type: 'SUBMISSIONS_ERROR',
          assignmentId,
          payload: 'Failed to load submissions.',
        });
      }
    },
    [],
  );

  const pollComposited = useCallback(async (submissionId: string) => {
    try {
      const data = await fetchCompositedDocument(submissionId);
      dispatch({ type: 'COMPOSITED_LOADED', submissionId, payload: data });
      if (data.status === 'pending' || data.status === 'generating') {
        pollTimers.current[submissionId] = setTimeout(
          () => pollComposited(submissionId),
          POLL_INTERVAL_MS,
        );
      } else {
        delete pollTimers.current[submissionId];
      }
    } catch {
      dispatch({
        type: 'COMPOSITED_LOADED',
        submissionId,
        payload: { status: 'pending' },
      });
      pollTimers.current[submissionId] = setTimeout(
        () => pollComposited(submissionId),
        POLL_INTERVAL_MS,
      );
    }
  }, []);

  const startPollingComposited = useCallback(
    (submissionId: string) => {
      if (pollTimers.current[submissionId]) {
        return;
      }
      pollComposited(submissionId);
    },
    [pollComposited],
  );

  const stopPollingComposited = useCallback((submissionId: string) => {
    const timer = pollTimers.current[submissionId];
    if (timer) {
      clearTimeout(timer);
      delete pollTimers.current[submissionId];
    }
  }, []);

  const value = useMemo(
    () => ({
      submissionsByAssignmentId: state.submissionsByAssignmentId,
      submissionsStatusByAssignmentId: state.submissionsStatusByAssignmentId,
      submissionsErrorByAssignmentId: state.submissionsErrorByAssignmentId,
      loadSubmissions,
      compositedBySubmissionId: state.compositedBySubmissionId,
      startPollingComposited,
      stopPollingComposited,
    }),
    [state, loadSubmissions, startPollingComposited, stopPollingComposited],
  );

  return (
    <SubmissionsContext.Provider value={value}>
      {children}
    </SubmissionsContext.Provider>
  );
}

export function useSubmissions(): SubmissionsContextValue {
  const context = useContext(SubmissionsContext);
  if (!context) {
    throw new Error(
      'useSubmissions must be used within a SubmissionsProvider',
    );
  }
  return context;
}
