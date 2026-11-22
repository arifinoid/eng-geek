"use client";

import {
  useMutation,
  useQuery,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  createAssessment,
  createAttempt,
  getActivity,
  getDashboard,
  getGate,
  listAssessments,
  listProfiles,
  listUnitActivities,
  listUnits,
  type Activity,
  type Assessment,
  type Attempt,
  type CreateAssessmentResponse,
  type Dashboard,
  type Gate,
  type Profile,
  type Unit,
} from "@eng-geek/api-client";

/** Input for recording one attempt. */
export type RecordAttemptInput = {
  profileId: number;
  activityId: number;
  score: number;
  durationSec: number;
};

/** Turns a non-success envelope into an error so React Query sees a rejection. */
function reject(status: number): never {
  throw new Error(`request failed with status ${status}`);
}

export function useProfiles(): UseQueryResult<Profile[]> {
  return useQuery({
    queryKey: ["profiles"],
    queryFn: async () => {
      const response = await listProfiles({ householdId: 1 });
      if (response.status !== 200) reject(response.status);
      return response.data;
    },
  });
}

export function useUnits(profileId: number | undefined): UseQueryResult<Unit[]> {
  return useQuery({
    queryKey: ["units", profileId],
    queryFn: async () => {
      const response = await listUnits({ profileId: profileId as number });
      if (response.status !== 200) reject(response.status);
      return response.data;
    },
    enabled: profileId !== undefined,
  });
}

export function useDashboard(
  profileId: number | undefined,
  weeks = 8,
): UseQueryResult<Dashboard> {
  return useQuery({
    queryKey: ["dashboard", profileId, weeks],
    queryFn: async () => {
      const response = await getDashboard(profileId as number, { weeks });
      if (response.status !== 200) reject(response.status);
      return response.data;
    },
    enabled: profileId !== undefined,
  });
}

export function useUnitActivities(
  unitId: number | undefined,
): UseQueryResult<Activity[]> {
  return useQuery({
    queryKey: ["unit-activities", unitId],
    queryFn: async () => {
      const response = await listUnitActivities(unitId as number);
      if (response.status !== 200) reject(response.status);
      return response.data;
    },
    enabled: unitId !== undefined,
  });
}

export function useActivity(activityId: number | undefined): UseQueryResult<Activity> {
  return useQuery({
    queryKey: ["activity", activityId],
    queryFn: async () => {
      const response = await getActivity(activityId as number);
      if (response.status !== 200) reject(response.status);
      return response.data;
    },
    enabled: activityId !== undefined,
  });
}

export function useGate(
  profileId: number | undefined,
  level?: string,
): UseQueryResult<Gate> {
  return useQuery({
    queryKey: ["gate", profileId, level ?? null],
    queryFn: async () => {
      const response = await getGate(profileId as number, level ? { level } : undefined);
      if (response.status !== 200) reject(response.status);
      return response.data;
    },
    enabled: profileId !== undefined,
  });
}

export function useAssessments(
  profileId: number | undefined,
  limit = 20,
): UseQueryResult<Assessment[]> {
  return useQuery({
    queryKey: ["assessments", profileId, limit],
    queryFn: async () => {
      const response = await listAssessments(profileId as number, { limit });
      if (response.status !== 200) reject(response.status);
      return response.data;
    },
    enabled: profileId !== undefined,
  });
}

/** The four skill scores of one checkpoint or placement assessment. */
export type AssessmentScoresInput = {
  listening: number;
  reading: number;
  writing: number;
  speaking: number;
};

export type CreateAssessmentInput = {
  profileId: number;
  levelCode: string;
  kind: "checkpoint" | "placement";
  scores: AssessmentScoresInput;
};

export function useCreateAssessment(): UseMutationResult<
  CreateAssessmentResponse,
  Error,
  CreateAssessmentInput
> {
  return useMutation({
    mutationFn: async (input: CreateAssessmentInput) => {
      const response = await createAssessment({
        profileId: input.profileId,
        levelCode: input.levelCode,
        kind: input.kind,
        scores: input.scores,
      });
      if (response.status !== 201) throw new Error(`request failed (${response.status})`);
      return response.data;
    },
  });
}

export function useRecordAttempt(): UseMutationResult<
  Attempt,
  Error,
  RecordAttemptInput
> {
  return useMutation({
    mutationFn: async (input: RecordAttemptInput) => {
      const response = await createAttempt({
        profileId: input.profileId,
        activityId: input.activityId,
        score: input.score,
        durationSec: input.durationSec,
      });
      if (response.status !== 201) throw new Error(`request failed (${response.status})`);
      return response.data;
    },
  });
}
