import LoadingCircleIcon from "../icon/LoadingCircleIcon";

const Loading = () => {
  return (
    <div className="absolute top-0 right-0 bottom-0 left-0 flex w-full items-center justify-center">
      <LoadingCircleIcon />
    </div>
  );
};

export default Loading;
