import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { sortBy } from "lodash";

type OptionID = string;

export interface Question {
  question: string;
  why_this_matters: string;
  options: Record<OptionID, string>;
}

export type Questionnaire<T extends Record<string, Question>> = {
  [K in keyof T]: Question;
};

type SelectedOptions<T extends Record<string, Question>> = {
  [K in keyof T]?: OptionID[];
};

export const useQuestionnaire = <T extends Record<string, Question>>(questionnaire: T) => {
  const [searchParams, setSearchParams] = useSearchParams();

  const questionKeys = sortBy(
    Object.keys(questionnaire).filter((key) => /^question\d+$/.test(key)),
    (key) => parseInt(key.replace("question", ""))
  ) as (keyof T)[];

  const totalQuestions = questionKeys.length;

  const initialQParam = parseInt(searchParams.get("q") || "1", 10);
  const initialIndex =
    isNaN(initialQParam) || initialQParam < 1 || initialQParam > totalQuestions ? 0 : initialQParam - 1;

  const [currentIndex, setCurrentIndex] = useState(initialIndex);
  const [selectedOptions, setSelectedOptions] = useState<SelectedOptions<T>>({});

  const currentKey = questionKeys[currentIndex];
  const currentQuestion = questionnaire[currentKey];

  const updateQueryParam = (index: number) => {
    setSearchParams({ q: (index + 1).toString() }, { replace: false });
  };

  const toggleOption = (optionId: OptionID) => {
    setSelectedOptions((prev) => {
      const current = prev[currentKey] || [];
      const updated = current.includes(optionId) ? current.filter((id) => id !== optionId) : [...current, optionId];
      return {
        ...prev,
        [currentKey]: updated,
      };
    });
  };

  const next = () => {
    if (currentIndex < questionKeys.length - 1) {
      const nextIndex = currentIndex + 1;
      setCurrentIndex(nextIndex);
      updateQueryParam(nextIndex);
    }
  };

  const prev = () => {
    if (currentIndex > 0) {
      const prevIndex = currentIndex - 1;
      setCurrentIndex(prevIndex);
      updateQueryParam(prevIndex);
    }
  };

  // Keep index in sync when q changes manually (e.g. user clicks back)
  useEffect(() => {
    const qParam = parseInt(searchParams.get("q") || "1", 10);
    const index = isNaN(qParam) || qParam < 1 || qParam > totalQuestions ? 0 : qParam - 1;
    setCurrentIndex(index);
  }, [searchParams, totalQuestions]);

  const isFirst = currentIndex === 0;
  const isLast = currentIndex === questionKeys.length - 1;

  return {
    currentIndex: currentIndex + 1,
    currentKey,
    currentQuestion,
    selectedOptions,
    toggleOption,
    next,
    prev,
    isFirst,
    isLast,
    questionKeys,
    totalQuestions,
  };
};
