import TipBadge from "@/components/badge/TipBadge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function UserProfile() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center space-x-4">
        <Avatar className="h-20 w-20">
          <AvatarImage src="/avatar.jpg" alt="User's avatar" />
          <AvatarFallback>JD</AvatarFallback>
        </Avatar>
        <div>
          <CardTitle className="flex items-center gap-2 text-2xl">
            John Doe
            <TipBadge />
          </CardTitle>
          <p className="text-sm text-muted-foreground">@johndoe</p>
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <p className="font-semibold">导师</p>
            <p>Prof. Jane Smith</p>
          </div>
          <div>
            <p className="font-semibold">课题组</p>
            <p>AI Research Lab</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
